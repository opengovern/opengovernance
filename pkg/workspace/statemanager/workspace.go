package statemanager

import (
	"context"
	"encoding/json"
	"fmt"
	apimeta "github.com/fluxcd/pkg/apis/meta"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
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

	case api.StatusProvisioning:
	case api.StatusProvisioningFailed:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
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
		creds, err := s.db.ListCredentialsByWorkspace(workspace.Name)
		if err != nil {
			return fmt.Errorf("listing credentials: %w", err)
		}
		for _, cred := range creds {
			err = s.db.DeleteCredential(cred.ID)
			if err != nil {
				return fmt.Errorf("deleting credentials: %w", err)
			}
		}

		helmRelease, err := s.findHelmRelease(ctx, workspace)
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

		if err := s.db.DeleteWorkspace(workspace.ID); err != nil {
			return fmt.Errorf("update workspace status: %w", err)
		}
	case api.StatusSuspending:
		helmRelease, err := s.findHelmRelease(ctx, workspace)
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

		values := helmRelease.GetValues()
		currentReplicaCount, err := getReplicaCount(values)
		if err != nil {
			return fmt.Errorf("getReplicaCount: %w", err)
		}

		if currentReplicaCount != 0 {
			values, err = updateValuesSetReplicaCount(values, 0)
			if err != nil {
				return fmt.Errorf("updateValuesSetReplicaCount: %w", err)
			}

			b, err := json.Marshal(values)
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

	helmRelease, err := s.findHelmRelease(ctx, workspace)
	if err != nil {
		return fmt.Errorf("find helm release: %w", err)
	}
	if helmRelease == nil {
		s.logger.Info(fmt.Sprintf("create helm release %s with status %s", workspace.ID, workspace.Status))
		if err := s.createHelmRelease(ctx, workspace); err != nil {
			return fmt.Errorf("create helm release: %w", err)
		}
		// update the workspace status next loop
		return nil
	}

	values := helmRelease.GetValues()
	currentReplicaCount, err := getReplicaCount(values)
	if err != nil {
		return fmt.Errorf("getReplicaCount: %w", err)
	}

	if currentReplicaCount == 0 {
		values, err = updateValuesSetReplicaCount(values, 1)
		if err != nil {
			return fmt.Errorf("updateValuesSetReplicaCount: %w", err)
		}

		b, err := json.Marshal(values)
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

	newStatus := workspace.Status
	// check the status of helm release
	if meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition) {
		// when the helm release installed successfully, set the rolebinding
		limits := api.GetLimitsByTier(workspace.Tier)
		authCtx := &httpclient.Context{
			UserID:         workspace.OwnerId,
			UserRole:       authapi.AdminRole,
			WorkspaceName:  workspace.Name,
			WorkspaceID:    workspace.ID,
			MaxUsers:       limits.MaxUsers,
			MaxConnections: limits.MaxConnections,
			MaxResources:   limits.MaxResources,
		}

		if err := s.authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
			UserID:   workspace.OwnerId,
			RoleName: authapi.AdminRole,
		}); err != nil {
			return fmt.Errorf("put role binding: %w", err)
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

func (s *Service) addCredentialToWorkspace(workspaceID string, cred db.Credential) error {
	onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", workspaceID)
	onboardClient := client.NewOnboardServiceClient(onboardURL, s.cache)

	credential, err := onboardClient.PostCredentials(&httpclient.Context{UserRole: authapi.InternalRole}, api2.CreateCredentialRequest{
		SourceType: cred.ConnectorType,
		Config:     cred.Metadata,
	})
	if err != nil {
		return err
	}

	_, err = onboardClient.AutoOnboard(&httpclient.Context{UserRole: authapi.InternalRole}, credential.ID)
	if err != nil {
		return err
	}

	err = s.db.DeleteCredential(cred.ID)
	if err != nil {
		return err
	}

	return nil
}
