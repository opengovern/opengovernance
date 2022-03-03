package onboard

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/smithy-go"
	keibiaws "gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/describer"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

func discoverAwsAccounts(ctx context.Context, req api.DiscoverAWSAccountsRequest) ([]api.DiscoverAWSAccountsResponse, error) {
	cfg, err := keibiaws.GetConfig(ctx, req.AccessKey, req.SecretKey, "", "")
	if err != nil {
		return nil, err
	}

	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	accounts, err := describer.OrganizationAccounts(ctx, cfg)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}

		acc, err := currentAwsAccount(ctx, cfg)
		if err != nil {
			return nil, err
		}

		return []api.DiscoverAWSAccountsResponse{acc}, nil
	}

	org, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return nil, err
		}
	}

	var discovered []api.DiscoverAWSAccountsResponse
	for _, acc := range accounts {
		discovered = append(discovered, api.DiscoverAWSAccountsResponse{
			AccountID:      *acc.Id,
			Status:         string(acc.Status),
			Name:           *acc.Name,
			Email:          *acc.Email,
			OrganizationID: *org.Id,
		})
	}

	return discovered, nil
}

func currentAwsAccount(ctx context.Context, cfg aws.Config) (api.DiscoverAWSAccountsResponse, error) {
	accID, err := describer.STSAccount(ctx, cfg)
	if err != nil {
		return api.DiscoverAWSAccountsResponse{}, err
	}

	var (
		orgId    string
		accName  string
		accEmail string
	)
	orgs, err := describer.OrganizationOrganization(ctx, cfg)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return api.DiscoverAWSAccountsResponse{}, err
		}
	} else {
		orgId = *orgs.Id
	}

	acc, err := describer.OrganizationAccount(ctx, cfg, accID)
	if err != nil {
		if !ignoreAwsOrgError(err) {
			return api.DiscoverAWSAccountsResponse{}, err
		}
	} else {
		accName = *acc.Name
		accEmail = *acc.Email
	}

	return api.DiscoverAWSAccountsResponse{
		AccountID:      accID,
		Status:         string(types.AccountStatusActive),
		OrganizationID: orgId,
		Name:           accName,
		Email:          accEmail,
	}, nil
}

func ignoreAwsOrgError(err error) bool {
	var ae smithy.APIError
	return errors.As(err, &ae) &&
		(ae.ErrorCode() == (&types.AWSOrganizationsNotInUseException{}).ErrorCode() ||
			ae.ErrorCode() == (&types.AccessDeniedException{}).ErrorCode())
}
