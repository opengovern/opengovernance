package statemanager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kms2 "github.com/aws/aws-sdk-go/service/kms"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	apimeta "github.com/fluxcd/pkg/apis/meta"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

func (s *Service) handleWorkspace(workspace *db.Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := workspace.Status
	switch status {
	case api.StatusBootstrapping:
		err := s.runBootstrapping(workspace)
		if err != nil {
			return err
		}

	case api.StatusReserved:
		err := s.createWorkspace(workspace)
		if err != nil {
			return err
		}

	case api.StatusProvisioning:
	case api.StatusProvisioningFailed:
		helmRelease, err := s.FindHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease == nil {
			return nil
		}

		newStatus := status
		// check the status of helm release
		if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			newStatus = api.StatusProvisioning
		} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
			newStatus = api.StatusProvisioning
		} else if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
			newStatus = api.StatusProvisioningFailed
		}
		if newStatus != status {
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, newStatus); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	case api.StatusDeleting:
		creds, err := s.db.ListCredentialsByWorkspaceID(workspace.ID)
		if err != nil {
			return fmt.Errorf("listing credentials: %w", err)
		}
		for _, cred := range creds {
			err = s.db.DeleteCredential(cred.ID)
			if err != nil {
				return fmt.Errorf("deleting credentials: %w", err)
			}
		}

		err = s.deleteInsightBucket(ctx, workspace)
		if err != nil {
			return fmt.Errorf("deleting insight bucket: %w", err)
		}

		helmRelease, err := s.FindHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}

		if helmRelease != nil {
			s.logger.Info(fmt.Sprintf("delete helm release %s with status %s", workspace.ID, workspace.Status))
			if err := s.deleteHelmRelease(ctx, workspace); err != nil {
				return fmt.Errorf("delete helm release: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		namespace, err := s.findTargetNamespace(ctx, workspace.ID)
		if err != nil {
			return fmt.Errorf("find target namespace: %w", err)
		}
		if namespace != nil {
			s.logger.Info(fmt.Sprintf("delete target namespace %s with status %s", workspace.ID, workspace.Status))
			if err := s.deleteTargetNamespace(ctx, workspace.ID); err != nil {
				return fmt.Errorf("delete target namespace: %w", err)
			}
			// update the workspace status next loop
			return nil
		}

		if workspace.AWSUniqueId != nil {
			userName := fmt.Sprintf("kaytu-user-%s", *workspace.AWSUniqueId)
			iamClient := iam.NewFromConfig(s.awsMasterConfig)
			accessKeys, err := iamClient.ListAccessKeys(context.Background(), &iam.ListAccessKeysInput{
				UserName: aws.String(userName),
			})
			if err != nil {
				return err
			}
			for _, accessKey := range accessKeys.AccessKeyMetadata {
				_, err := iamClient.DeleteAccessKey(context.Background(), &iam.DeleteAccessKeyInput{
					UserName:    aws.String(userName),
					AccessKeyId: accessKey.AccessKeyId,
				})
				if err != nil {
					return err
				}
			}

			policies, err := iamClient.ListAttachedUserPolicies(context.Background(), &iam.ListAttachedUserPoliciesInput{
				UserName: aws.String(userName),
			})
			if err != nil {
				return err
			}

			for _, policy := range policies.AttachedPolicies {
				_, err = iamClient.DetachUserPolicy(context.Background(), &iam.DetachUserPolicyInput{
					UserName:  aws.String(userName),
					PolicyArn: policy.PolicyArn,
				})
				if err != nil {
					return err
				}

				_, err = iamClient.DeleteUserPolicy(context.Background(), &iam.DeleteUserPolicyInput{
					PolicyName: policy.PolicyName,
					UserName:   aws.String(userName),
				})
				if err != nil {
					return err
				}
			}

			_, err = iamClient.DeleteUser(context.Background(), &iam.DeleteUserInput{
				UserName: aws.String(userName),
			})
			if err != nil {
				return err
			}

			err = s.db.DeleteMasterCredential(*workspace.AWSUniqueId)
			if err != nil {
				return err
			}
		}

		if err := s.db.DeleteWorkspace(workspace.ID); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	case api.StatusSuspending:
		helmRelease, err := s.FindHelmRelease(ctx, workspace)
		if err != nil {
			return fmt.Errorf("find helm release: %w", err)
		}
		if helmRelease == nil {
			return fmt.Errorf("cannot find helmrelease")
		}

		var pods corev1.PodList
		err = s.kubeClient.List(ctx, &pods, k8sclient.InNamespace(workspace.ID))
		if err != nil {
			return fmt.Errorf("fetching list of pods: %w", err)
		}

		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, "describe-connection-worker") {
				// waiting for describe jobs to finish
				return nil
			}
		}

		settings, err := GetWorkspaceHelmValues(helmRelease)
		if err != nil {
			return fmt.Errorf("getReplicaCount: %w", err)
		}

		if settings.Kaytu.ReplicaCount != 0 {
			settings.Kaytu.ReplicaCount = 0
			err = s.UpdateWorkspaceSettings(helmRelease, settings)
			if err != nil {
				return err
			}

			return nil
		}

		if len(pods.Items) > 0 {
			// waiting for pods to go down
			return nil
		}

		if err := s.db.UpdateWorkspaceStatus(workspace.ID, api.StatusSuspended); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	}
	return nil
}

func (s *Service) createWorkspace(workspace *db.Workspace) error {
	ctx := context.Background()

	helmRelease, err := s.FindHelmRelease(ctx, workspace)
	if err != nil {
		return fmt.Errorf("find helm release: %w", err)
	}
	if helmRelease == nil {
		if workspace.OwnerId != nil {
			rs, err := s.db.GetReservedWorkspace()
			if err != nil {
				return err
			}

			if rs != nil {
				err = s.db.DeleteWorkspace(workspace.ID)
				if err != nil {
					return err
				}

				err = s.db.UpdateCredentialWSID(workspace.ID, rs.ID)
				if err != nil {
					return err
				}

				workspace.ID = rs.ID
				if err := s.db.UpdateWorkspace(workspace); err != nil {
					return err
				}

				limits := api.GetLimitsByTier(workspace.Tier)
				authCtx := &httpclient.Context{
					UserID:         *workspace.OwnerId,
					UserRole:       authapi.AdminRole,
					WorkspaceName:  workspace.Name,
					WorkspaceID:    workspace.ID,
					MaxUsers:       limits.MaxUsers,
					MaxConnections: limits.MaxConnections,
					MaxResources:   limits.MaxResources,
				}

				if err := s.authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
					UserID:   *workspace.OwnerId,
					RoleName: authapi.AdminRole,
				}); err != nil {
					return fmt.Errorf("put role binding: %w", err)
				}

				helmRelease, err := s.FindHelmRelease(context.Background(), workspace)
				if err != nil {
					return fmt.Errorf("find helm release: %w", err)
				}
				if helmRelease == nil {
					return fmt.Errorf("helm release not found")
				}

				settings, err := GetWorkspaceHelmValues(helmRelease)
				if err != nil {
					return err
				}

				settings.Kaytu.Workspace.Name = workspace.Name
				if workspace.AWSUserARN != nil {
					settings.Kaytu.Workspace.UserARN = *workspace.AWSUserARN
				}
				if workspace.AWSUniqueId != nil {
					masterCred, err := s.db.GetMasterCredentialByWorkspaceUID(*workspace.AWSUniqueId)
					if err != nil {
						return err
					}

					decoded, err := base64.StdEncoding.DecodeString(masterCred.Credential)
					if err != nil {
						return err
					}

					result, err := s.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
						CiphertextBlob:      decoded,
						EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
						KeyId:               &s.cfg.KMSKeyARN,
						EncryptionContext:   nil, //TODO-Saleh use workspaceID
					})
					if err != nil {
						return fmt.Errorf("failed to encrypt ciphertext: %v", err)
					}

					var accessKey types.AccessKey
					err = json.Unmarshal(result.Plaintext, &accessKey)
					//err = json.Unmarshal([]byte(masterCred.Credential), &accessKey)
					if err != nil {
						return err
					}

					settings.Kaytu.Workspace.MasterAccessKey = *accessKey.AccessKeyId
					settings.Kaytu.Workspace.MasterSecretKey = *accessKey.SecretAccessKey
				}
				err = s.UpdateWorkspaceSettings(helmRelease, settings)
				if err != nil {
					return err
				}

				var res corev1.PodList
				err = s.kubeClient.List(context.Background(), &res)
				if err != nil {
					return fmt.Errorf("listing pods: %w", err)
				}
				for _, pod := range res.Items {
					if strings.HasPrefix(pod.Name, "describe-scheduler") {
						err = s.kubeClient.Delete(context.Background(), &pod)
						if err != nil {
							return fmt.Errorf("deleting pods: %w", err)
						}
					}
				}

				return nil
			}
		}

		s.logger.Info(fmt.Sprintf("create helm release %s with status %s", workspace.ID, workspace.Status))
		if err := s.createHelmRelease(ctx, workspace); err != nil {
			return fmt.Errorf("create helm release: %w", err)
		}
		// update the workspace status next loop
		return nil
	}

	settings, err := GetWorkspaceHelmValues(helmRelease)
	if err != nil {
		return fmt.Errorf("getReplicaCount: %w", err)
	}

	if settings.Kaytu.ReplicaCount == 0 {
		settings.Kaytu.ReplicaCount = 1
		err = s.UpdateWorkspaceSettings(helmRelease, settings)
		if err != nil {
			return err
		}

		return nil
	}

	newStatus := workspace.Status
	// check the status of helm release
	if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
		// when the helm release installed successfully, set the rolebinding
		limits := api.GetLimitsByTier(workspace.Tier)
		if workspace.OwnerId != nil {
			authCtx := &httpclient.Context{
				UserID:         *workspace.OwnerId,
				UserRole:       authapi.AdminRole,
				WorkspaceName:  workspace.Name,
				WorkspaceID:    workspace.ID,
				MaxUsers:       limits.MaxUsers,
				MaxConnections: limits.MaxConnections,
				MaxResources:   limits.MaxResources,
			}

			if err := s.authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
				UserID:   *workspace.OwnerId,
				RoleName: authapi.AdminRole,
			}); err != nil {
				return fmt.Errorf("put role binding: %w", err)
			}
		}

		err = s.rdb.SetEX(context.Background(), "last_access_"+workspace.Name, time.Now().UnixMilli(), time.Duration(s.cfg.AutoSuspendDurationMinutes)*time.Minute).Err()
		if err != nil {
			return fmt.Errorf("set last access: %v", err)
		}

		err = s.db.SetWorkspaceCreated(workspace.ID)
		if err != nil {
			return fmt.Errorf("set last access: %v", err)
		}
		return nil
	} else if meta.IsStatusConditionFalse(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
		if !helmRelease.Spec.Suspend {
			helmRelease.Spec.Suspend = true
			err = s.kubeClient.Update(ctx, helmRelease)
			if err != nil {
				return fmt.Errorf("suspend helmrelease: %v", err)
			}
		} else {
			helmRelease.Spec.Suspend = false
			err = s.kubeClient.Update(ctx, helmRelease)
			if err != nil {
				return fmt.Errorf("suspend helmrelease: %v", err)
			}
		}
	} else if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.StalledCondition) {
		newStatus = api.StatusProvisioningFailed
		if newStatus != workspace.Status {
			if err := s.db.UpdateWorkspaceStatus(workspace.ID, newStatus); err != nil {
				return fmt.Errorf("update workspace status: %w", err)
			}
		}
	}
	return nil
}

func (s *Service) addCredentialToWorkspace(workspace *db.Workspace, cred db.Credential) error {
	onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
	onboardClient := client.NewOnboardServiceClient(onboardURL, s.cache)

	var request api.AddCredentialRequest
	decoded, err := base64.StdEncoding.DecodeString(cred.Metadata)
	if err != nil {
		return err
	}

	result, err := s.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
		CiphertextBlob:      decoded,
		EncryptionAlgorithm: kms2.EncryptionAlgorithmSpecSymmetricDefault,
		KeyId:               &s.cfg.KMSKeyARN,
		EncryptionContext:   nil, //TODO-Saleh use workspaceID
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt ciphertext: %v", err)
	}

	err = json.Unmarshal(result.Plaintext, &request)
	if err != nil {
		return err
	}

	limits := api.GetLimitsByTier(workspace.Tier)
	if cred.ConnectorType == source.CloudAWS {
		if cred.SingleConnection {
			_, err := onboardClient.PostConnectionAws(&httpclient.Context{UserRole: authapi.InternalRole, MaxConnections: limits.MaxConnections}, api2.CreateAwsConnectionRequest{
				Name:      "",
				AWSConfig: request.AWSConfig,
			})
			if err != nil {
				return err
			}
		} else {
			credential, err := onboardClient.CreateCredentialV2(&httpclient.Context{UserRole: authapi.InternalRole, MaxConnections: limits.MaxConnections}, apiv2.CreateCredentialV2Request{
				Connector: cred.ConnectorType,
				AWSConfig: request.AWSConfig,
			})
			if err != nil {
				return err
			}

			_, err = onboardClient.AutoOnboard(&httpclient.Context{UserRole: authapi.InternalRole, MaxConnections: limits.MaxConnections}, credential.ID)
			if err != nil {
				return err
			}
		}
	} else {
		credential, err := onboardClient.PostCredentials(&httpclient.Context{UserRole: authapi.InternalRole, MaxConnections: limits.MaxConnections}, api2.CreateCredentialRequest{
			SourceType: cred.ConnectorType,
			Config:     request.AzureConfig,
		})
		if err != nil {
			return err
		}

		_, err = onboardClient.AutoOnboard(&httpclient.Context{UserRole: authapi.InternalRole, MaxConnections: limits.MaxConnections}, credential.ID)
		if err != nil {
			return err
		}
	}

	err = s.db.SetIsCreated(cred.ID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateWorkspaceSettings(helmRelease *v2beta1.HelmRelease, settings KaytuWorkspaceSettings) error {
	ctx := context.Background()
	b, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshalling values: %w", err)
	}
	helmRelease.Spec.Values.Raw = b
	err = s.kubeClient.Update(ctx, helmRelease)
	if err != nil {
		return fmt.Errorf("updating replica count: %w", err)
	}
	return nil
}

func GetWorkspaceHelmValues(helmRelease *v2beta1.HelmRelease) (KaytuWorkspaceSettings, error) {
	var settings KaytuWorkspaceSettings

	values := helmRelease.GetValues()
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return settings, err
	}

	err = json.Unmarshal(valuesJSON, &settings)
	if err != nil {
		return settings, err
	}

	return settings, nil
}
