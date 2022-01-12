package describer

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/gocarina/gocsv"
)

const (
	organizationsNotInUseException = "AWSOrganizationsNotInUseException"
)

type IAMAccountDescription struct {
	Aliases      []string
	Organization *orgtypes.Organization
}

func IAMAccount(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	orgClient := organizations.NewFromConfig(cfg)

	output, err := orgClient.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		if isErr(err, organizationsNotInUseException) {
			output = &organizations.DescribeOrganizationOutput{}
		} else {
			return nil, err
		}
	}

	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListAccountAliasesPaginator(client, &iam.ListAccountAliasesInput{})

	var aliases []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		aliases = append(aliases, page.AccountAliases...)
	}

	return []Resource{
		{
			// No ID or ARN. Per Account Configuration
			Description: IAMAccountDescription{
				Aliases:      aliases,
				Organization: output.Organization,
			},
		},
	}, nil
}

type AccountSummary struct {
	AccountMFAEnabled                 int32
	AccessKeysPerUserQuota            int32
	AccountAccessKeysPresent          int32
	AccountSigningCertificatesPresent int32
	AssumeRolePolicySizeQuota         int32
	AttachedPoliciesPerGroupQuota     int32
	AttachedPoliciesPerRoleQuota      int32
	AttachedPoliciesPerUserQuota      int32
	GlobalEndpointTokenVersion        int32
	GroupPolicySizeQuota              int32
	Groups                            int32
	GroupsPerUserQuota                int32
	GroupsQuota                       int32
	InstanceProfiles                  int32
	InstanceProfilesQuota             int32
	MFADevices                        int32
	MFADevicesInUse                   int32
	Policies                          int32
	PoliciesQuota                     int32
	PolicySizeQuota                   int32
	PolicyVersionsInUse               int32
	PolicyVersionsInUseQuota          int32
	Providers                         int32
	RolePolicySizeQuota               int32
	Roles                             int32
	RolesQuota                        int32
	ServerCertificates                int32
	ServerCertificatesQuota           int32
	SigningCertificatesPerUserQuota   int32
	UserPolicySizeQuota               int32
	Users                             int32
	UsersQuota                        int32
	VersionsPerPolicyQuota            int32
}

type IAMAccountSummaryDescription struct {
	AccountSummary AccountSummary
}

func IAMAccountSummary(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.GetAccountSummary(ctx, &iam.GetAccountSummaryInput{})
	if err != nil {
		return nil, err
	}

	desc := IAMAccountSummaryDescription{
		AccountSummary: AccountSummary{
			AccountMFAEnabled:                 output.SummaryMap["AccountMFAEnabled"],
			AccessKeysPerUserQuota:            output.SummaryMap["AccessKeysPerUserQuota"],
			AccountAccessKeysPresent:          output.SummaryMap["AccountAccessKeysPresent"],
			AccountSigningCertificatesPresent: output.SummaryMap["AccountSigningCertificatesPresent"],
			AssumeRolePolicySizeQuota:         output.SummaryMap["AssumeRolePolicySizeQuota"],
			AttachedPoliciesPerGroupQuota:     output.SummaryMap["AttachedPoliciesPerGroupQuota"],
			AttachedPoliciesPerRoleQuota:      output.SummaryMap["AttachedPoliciesPerRoleQuota"],
			AttachedPoliciesPerUserQuota:      output.SummaryMap["AttachedPoliciesPerUserQuota"],
			GlobalEndpointTokenVersion:        output.SummaryMap["GlobalEndpointTokenVersion"],
			GroupPolicySizeQuota:              output.SummaryMap["GroupPolicySizeQuota"],
			Groups:                            output.SummaryMap["Groups"],
			GroupsPerUserQuota:                output.SummaryMap["GroupsPerUserQuota"],
			GroupsQuota:                       output.SummaryMap["GroupsQuota"],
			InstanceProfiles:                  output.SummaryMap["InstanceProfiles"],
			InstanceProfilesQuota:             output.SummaryMap["InstanceProfilesQuota"],
			MFADevices:                        output.SummaryMap["MFADevices"],
			MFADevicesInUse:                   output.SummaryMap["MFADevicesInUse"],
			Policies:                          output.SummaryMap["Policies"],
			PoliciesQuota:                     output.SummaryMap["PoliciesQuota"],
			PolicySizeQuota:                   output.SummaryMap["PolicySizeQuota"],
			PolicyVersionsInUse:               output.SummaryMap["PolicyVersionsInUse"],
			PolicyVersionsInUseQuota:          output.SummaryMap["PolicyVersionsInUseQuota"],
			Providers:                         output.SummaryMap["Providers"],
			RolePolicySizeQuota:               output.SummaryMap["RolePolicySizeQuota"],
			Roles:                             output.SummaryMap["Roles"],
			RolesQuota:                        output.SummaryMap["RolesQuota"],
			ServerCertificates:                output.SummaryMap["ServerCertificates"],
			ServerCertificatesQuota:           output.SummaryMap["ServerCertificatesQuota"],
			SigningCertificatesPerUserQuota:   output.SummaryMap["SigningCertificatesPerUserQuota"],
			UserPolicySizeQuota:               output.SummaryMap["UserPolicySizeQuota"],
			Users:                             output.SummaryMap["Users"],
			UsersQuota:                        output.SummaryMap["UsersQuota"],
			VersionsPerPolicyQuota:            output.SummaryMap["VersionsPerPolicyQuota"],
		},
	}
	return []Resource{
		{
			// No ID or ARN. Per Account Configuration
			Description: desc,
		},
	}, nil
}

type IAMAccessKeyDescription struct {
	AccessKey types.AccessKeyMetadata
}

func IAMAccessKey(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListAccessKeysPaginator(client, &iam.ListAccessKeysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AccessKeyMetadata {
			values = append(values, Resource{
				ID: *v.AccessKeyId,
				Description: IAMAccessKeyDescription{
					AccessKey: v,
				},
			})
		}
	}

	return values, nil
}

type CredentialReport struct {
	GeneratedTime             *time.Time `csv:"-"`
	UserArn                   string     `csv:"arn"`
	UserName                  string     `csv:"user"`
	UserCreationTime          string     `csv:"user_creation_time"`
	AccessKey1Active          bool       `csv:"access_key_1_active"`
	AccessKey1LastRotated     string     `csv:"access_key_1_last_rotated"`
	AccessKey1LastUsedDate    string     `csv:"access_key_1_last_used_date"`
	AccessKey1LastUsedRegion  string     `csv:"access_key_1_last_used_region"`
	AccessKey1LastUsedService string     `csv:"access_key_1_last_used_service"`
	AccessKey2Active          bool       `csv:"access_key_2_active"`
	AccessKey2LastRotated     string     `csv:"access_key_2_last_rotated"`
	AccessKey2LastUsedDate    string     `csv:"access_key_2_last_used_date"`
	AccessKey2LastUsedRegion  string     `csv:"access_key_2_last_used_region"`
	AccessKey2LastUsedService string     `csv:"access_key_2_last_used_service"`
	Cert1Active               bool       `csv:"cert_1_active"`
	Cert1LastRotated          string     `csv:"cert_1_last_rotated"`
	Cert2Active               bool       `csv:"cert_2_active"`
	Cert2LastRotated          string     `csv:"cert_2_last_rotated"`
	MFAActive                 bool       `csv:"mfa_active"`
	PasswordEnabled           string     `csv:"password_enabled"`
	PasswordLastChanged       string     `csv:"password_last_changed"`
	PasswordLastUsed          string     `csv:"password_last_used"`
	PasswordNextRotation      string     `csv:"password_next_rotation"`
}

type IAMCredentialReportDescription struct {
	CredentialReport CredentialReport
}

func IAMCredentialReport(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.GetCredentialReport(ctx, &iam.GetCredentialReportInput{})
	if err != nil {
		if isErr(err, (&types.CredentialReportNotPresentException{}).ErrorCode()) ||
			isErr(err, (&types.CredentialReportExpiredException{}).ErrorCode()) ||
			isErr(err, (&types.CredentialReportNotPresentException{}).ErrorCode()) {
			return nil, nil
		}
		return nil, err
	}

	reports := []CredentialReport{}
	if err := gocsv.UnmarshalString(string(output.Content), &reports); err != nil {
		return nil, err
	}

	var values []Resource
	for _, report := range reports {
		report.GeneratedTime = output.GeneratedTime
		values = append(values, Resource{
			ID: report.UserName, // Unique report entry per user
			Description: IAMCredentialReportDescription{
				CredentialReport: report,
			},
		})
	}

	return values, nil
}

type IAMPolicyDescription struct {
	Policy        types.Policy
	PolicyVersion types.PolicyVersion
}

func IAMPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{
		OnlyAttached: false,
		Scope:        types.PolicyScopeTypeAll,
	})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Policies {
			version, err := client.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
				PolicyArn: v.Arn,
				VersionId: v.DefaultVersionId,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN: *v.Arn,
				Description: IAMPolicyDescription{
					Policy:        v,
					PolicyVersion: *version.PolicyVersion,
				},
			})
		}
	}

	return values, nil
}

type IAMGroupDescription struct {
	Group              types.Group
	Users              []types.User
	InlinePolicies     []InlinePolicy
	AttachedPolicyArns []string
}

func IAMGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListGroupsPaginator(client, &iam.ListGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Groups {
			users, err := getGroupUsers(ctx, client, v.GroupName)
			if err != nil {
				return nil, err
			}

			policies, err := getGroupPolicies(ctx, client, v.GroupName)
			if err != nil {
				return nil, err
			}

			aPolicies, err := getGroupAttachedPolicyArns(ctx, client, v.GroupName)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN: *v.Arn,
				Description: IAMGroupDescription{
					Group:              v,
					Users:              users,
					InlinePolicies:     policies,
					AttachedPolicyArns: aPolicies,
				},
			})
		}
	}

	return values, nil
}

func getGroupUsers(ctx context.Context, client *iam.Client, groupname *string) ([]types.User, error) {
	paginator := iam.NewGetGroupPaginator(client, &iam.GetGroupInput{
		GroupName: groupname,
	})

	var users []types.User
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		users = append(users, page.Users...)
	}

	return users, nil
}

func getGroupPolicies(ctx context.Context, client *iam.Client, groupname *string) ([]InlinePolicy, error) {
	paginator := iam.NewListGroupPoliciesPaginator(client, &iam.ListGroupPoliciesInput{
		GroupName: groupname,
	})

	var policies []InlinePolicy
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.PolicyNames {
			output, err := client.GetGroupPolicy(ctx, &iam.GetGroupPolicyInput{
				PolicyName: aws.String(p),
				GroupName:  groupname,
			})
			if err != nil {
				return nil, err
			}

			policies = append(policies, InlinePolicy{
				PolicyName:     *output.PolicyName,
				PolicyDocument: *output.PolicyDocument,
			})
		}
	}

	return policies, nil
}

func getGroupAttachedPolicyArns(ctx context.Context, client *iam.Client, groupname *string) ([]string, error) {
	paginator := iam.NewListAttachedGroupPoliciesPaginator(client, &iam.ListAttachedGroupPoliciesInput{
		GroupName: groupname,
	})

	var arns []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.AttachedPolicies {
			arns = append(arns, *p.PolicyArn)

		}
	}

	return arns, nil
}

func IAMInstanceProfile(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListInstanceProfilesPaginator(client, &iam.ListInstanceProfilesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.InstanceProfiles {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMManagedPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{
		OnlyAttached: true,
	})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Policies {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Description: v,
			})
		}
	}

	return values, nil
}

func IAMOIDCProvider(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.OpenIDConnectProviderList {
		values = append(values, Resource{
			ARN:         *v.Arn,
			Description: v,
		})
	}

	return values, nil
}

func IAMGroupPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	groups, err := IAMGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []Resource
	for _, g := range groups {
		group := g.Description.(types.Group)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListGroupPolicies(ctx, &iam.ListGroupPoliciesInput{
				GroupName: group.GroupName,
				Marker:    prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				v, err := client.GetGroupPolicy(ctx, &iam.GetGroupPolicyInput{
					GroupName:  group.GroupName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID:          CompositeID(*v.GroupName, *v.PolicyName),
					Description: v,
				})
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMUserPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	users, err := IAMUser(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []Resource
	for _, u := range users {
		user := u.Description.(types.User)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{
				UserName: user.UserName,
				Marker:   prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				v, err := client.GetUserPolicy(ctx, &iam.GetUserPolicyInput{
					UserName:   user.UserName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID:          CompositeID(*v.UserName, *v.PolicyName),
					Description: v,
				})
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func IAMRolePolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	roles, err := IAMRole(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(cfg)

	var values []Resource

	for _, r := range roles {
		role := r.Description.(types.Role)
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
				RoleName: role.RoleName,
				Marker:   prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, policy := range output.PolicyNames {
				v, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
					RoleName:   role.RoleName,
					PolicyName: aws.String(policy),
				})
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID:          CompositeID(*v.RoleName, *v.PolicyName),
					Description: v,
				})
			}

			return output.Marker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

type IAMRoleDescription struct {
	Role                types.Role
	InstanceProfileArns []string
	InlinePolicies      []InlinePolicy
	AttachedPolicyArns  []string
}

func IAMRole(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Roles {
			profiles, err := getRoleInstanceProfileArns(ctx, client, v.RoleName)
			if err != nil {
				return nil, err
			}

			policies, err := getRolePolicies(ctx, client, v.RoleName)
			if err != nil {
				return nil, err
			}

			aPolicies, err := getRoleAttachedPolicyArns(ctx, client, v.RoleName)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN: *v.Arn,
				Description: IAMRoleDescription{
					Role:                v,
					InstanceProfileArns: profiles,
					InlinePolicies:      policies,
					AttachedPolicyArns:  aPolicies,
				},
			})
		}
	}

	return values, nil
}

func getRoleInstanceProfileArns(ctx context.Context, client *iam.Client, rolename *string) ([]string, error) {
	paginator := iam.NewListInstanceProfilesForRolePaginator(client, &iam.ListInstanceProfilesForRoleInput{
		RoleName: rolename,
	})

	var arns []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, ip := range page.InstanceProfiles {
			arns = append(arns, *ip.Arn)

		}
	}

	return arns, nil
}

func getRolePolicies(ctx context.Context, client *iam.Client, rolename *string) ([]InlinePolicy, error) {
	paginator := iam.NewListRolePoliciesPaginator(client, &iam.ListRolePoliciesInput{
		RoleName: rolename,
	})

	var policies []InlinePolicy
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, name := range page.PolicyNames {
			output, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
				RoleName:   rolename,
				PolicyName: aws.String(name),
			})
			if err != nil {
				return nil, err
			}

			policies = append(policies, InlinePolicy{
				PolicyName:     *output.PolicyName,
				PolicyDocument: *output.PolicyDocument,
			})
		}

	}

	return policies, nil
}
func getRoleAttachedPolicyArns(ctx context.Context, client *iam.Client, rolename *string) ([]string, error) {
	paginator := iam.NewListAttachedRolePoliciesPaginator(client, &iam.ListAttachedRolePoliciesInput{
		RoleName: rolename,
	})

	var arns []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.AttachedPolicies {
			arns = append(arns, *p.PolicyArn)

		}
	}

	return arns, nil
}

func IAMSAMLProvider(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListSAMLProviders(ctx, &iam.ListSAMLProvidersInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.SAMLProviderList {
		values = append(values, Resource{
			ARN:         *v.Arn,
			Description: v,
		})
	}

	return values, nil
}

type IAMServerCertificateDescription struct {
	ServerCertificate types.ServerCertificate
}

func IAMServerCertificate(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListServerCertificatesPaginator(client, &iam.ListServerCertificatesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ServerCertificateMetadataList {
			output, err := client.GetServerCertificate(ctx, &iam.GetServerCertificateInput{
				ServerCertificateName: v.ServerCertificateName,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN: *v.Arn,
				Description: IAMServerCertificateDescription{
					ServerCertificate: *output.ServerCertificate,
				},
			})
		}
	}

	return values, nil
}

type InlinePolicy struct {
	PolicyName     string
	PolicyDocument string
}

type IAMUserDescription struct {
	User               types.User
	Groups             []types.Group
	InlinePolicies     []InlinePolicy
	AttachedPolicyArns []string
	MFADevices         []types.MFADevice
}

func IAMUser(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Users {
			policies, err := getUserPolicies(ctx, client, v.UserName)
			if err != nil {
				return nil, err
			}

			aPolicies, err := getUserAttachedPolicyArns(ctx, client, v.UserName)
			if err != nil {
				return nil, err
			}

			groups, err := getUserGroups(ctx, client, v.UserName)
			if err != nil {
				return nil, err
			}

			devices, err := getUserMFADevices(ctx, client, v.UserName)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN: *v.Arn,
				Description: IAMUserDescription{
					User:               v,
					Groups:             groups,
					InlinePolicies:     policies,
					AttachedPolicyArns: aPolicies,
					MFADevices:         devices,
				},
			})
		}
	}

	return values, nil
}

func getUserPolicies(ctx context.Context, client *iam.Client, username *string) ([]InlinePolicy, error) {
	paginator := iam.NewListUserPoliciesPaginator(client, &iam.ListUserPoliciesInput{
		UserName: username,
	})

	var policies []InlinePolicy
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.PolicyNames {
			output, err := client.GetUserPolicy(ctx, &iam.GetUserPolicyInput{
				PolicyName: aws.String(p),
				UserName:   username,
			})
			if err != nil {
				return nil, err
			}

			policies = append(policies, InlinePolicy{
				PolicyName:     *output.PolicyName,
				PolicyDocument: *output.PolicyDocument,
			})
		}
	}

	return policies, nil
}

func getUserAttachedPolicyArns(ctx context.Context, client *iam.Client, username *string) ([]string, error) {
	paginator := iam.NewListAttachedUserPoliciesPaginator(client, &iam.ListAttachedUserPoliciesInput{
		UserName: username,
	})

	var arns []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range page.AttachedPolicies {
			arns = append(arns, *p.PolicyArn)

		}
	}

	return arns, nil
}

func getUserGroups(ctx context.Context, client *iam.Client, username *string) ([]types.Group, error) {
	paginator := iam.NewListGroupsForUserPaginator(client, &iam.ListGroupsForUserInput{
		UserName: username,
	})

	var groups []types.Group
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		groups = append(groups, page.Groups...)
	}

	return groups, nil
}

func getUserMFADevices(ctx context.Context, client *iam.Client, username *string) ([]types.MFADevice, error) {
	paginator := iam.NewListMFADevicesPaginator(client, &iam.ListMFADevicesInput{
		UserName: username,
	})

	var devices []types.MFADevice
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		devices = append(devices, page.MFADevices...)
	}

	return devices, nil
}

func IAMVirtualMFADevice(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := iam.NewFromConfig(cfg)
	output, err := client.ListMFADevices(ctx, &iam.ListMFADevicesInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.MFADevices {
		values = append(values, Resource{
			ID:          *v.SerialNumber,
			Description: v,
		})
	}

	return values, nil
}
