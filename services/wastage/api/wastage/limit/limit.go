package limit

import (
	"context"
	"fmt"
	"github.com/kaytu-io/open-governance/services/wastage/db/repo"
	"go.uber.org/zap"
	"strings"
	"time"
)

type Service struct {
	logger    *zap.Logger
	userRepo  repo.UserRepo
	orgRepo   repo.OrganizationRepo
	usageRepo repo.UsageV2Repo
}

func NewLimitService(logger *zap.Logger, userRepo repo.UserRepo, orgRepo repo.OrganizationRepo, usageRepo repo.UsageV2Repo) *Service {
	return &Service{
		logger:    logger,
		userRepo:  userRepo,
		orgRepo:   orgRepo,
		usageRepo: usageRepo,
	}
}

func (s *Service) CheckRDSInstanceLimit(ctx context.Context, auth0UserId, orgEmail string) (bool, error) {
	s.logger.Info("Checking RDS Instance limit", zap.String("auth0UserId", auth0UserId), zap.String("orgEmail", orgEmail))
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgCount, err := s.usageRepo.GetRDSInstanceOptimizationsCountForOrg(ctx, org[1])
			if err != nil {
				return false, err
			}
			if orgCount < int64(OrgRDSInstanceLimit) {
				return true, nil
			}
			s.logger.Info("Org RDS Instance limit reached", zap.String("orgEmail", org[1]))
		}
	}
	userCount, err := s.usageRepo.GetRDSInstanceOptimizationsCountForUser(ctx, auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserRDSInstanceLimit) {
		return true, nil
	}
	s.logger.Info("User RDS Instance limit reached", zap.String("auth0UserId", auth0UserId))
	return false, nil
}

func (s *Service) CheckRDSClusterLimit(ctx context.Context, auth0UserId, orgEmail string) (bool, error) {
	s.logger.Info("Checking RDS Cluster limit", zap.String("auth0UserId", auth0UserId), zap.String("orgEmail", orgEmail))
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgCount, err := s.usageRepo.GetRDSClusterOptimizationsCountForOrg(ctx, org[1])
			if err != nil {
				return false, err
			}
			if orgCount < int64(OrgRDSClusterLimit) {
				return true, nil
			}
			s.logger.Info("Org RDS Cluster limit reached", zap.String("orgEmail", org[1]))
		}
	}
	userCount, err := s.usageRepo.GetRDSClusterOptimizationsCountForUser(ctx, auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserRDSClusterLimit) {
		return true, nil
	}
	s.logger.Info("User RDS Cluster limit reached", zap.String("auth0UserId", auth0UserId))
	return false, nil
}

func (s *Service) CheckPremiumAndSendErr(ctx context.Context, userId string, orgEmail string, service string) error {
	user, err := s.userRepo.Get(ctx, userId)
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err))
		return err
	}
	if user != nil && user.PremiumUntil != nil {
		if time.Now().Before(*user.PremiumUntil) {
			return nil
		}
	}

	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgName := strings.Split(orgEmail, "@")
			org, err := s.orgRepo.Get(ctx, orgName[1])
			if err != nil {
				s.logger.Error("failed to get organization", zap.Error(err))
				return err
			}
			if org != nil && org.PremiumUntil != nil {
				if time.Now().Before(*org.PremiumUntil) {
					return nil
				}
			}
		}
	}

	err = fmt.Errorf("reached the %s limit for both user and organization", service)
	s.logger.Error(err.Error(), zap.String("auth0UserId", userId), zap.String("orgEmail", orgEmail))
	return nil
}

func (s *Service) CheckEC2InstanceLimit(ctx context.Context, auth0UserId, orgEmail string) (bool, error) {
	s.logger.Info("Checking EC2 Instance limit", zap.String("auth0UserId", auth0UserId), zap.String("orgEmail", orgEmail))
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgCount, err := s.usageRepo.GetEC2InstanceOptimizationsCountForOrg(ctx, org[1])
			if err != nil {
				return false, err
			}
			if orgCount < int64(OrgEC2InstanceLimit) {
				return true, nil
			}
			s.logger.Info("Org EC2 Instance limit reached", zap.String("orgEmail", org[1]))
		}
	}
	userCount, err := s.usageRepo.GetEC2InstanceOptimizationsCountForUser(ctx, auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserEC2InstanceLimit) {
		return true, nil
	}
	s.logger.Info("User EC2 Instance limit reached", zap.String("auth0UserId", auth0UserId))
	return false, nil
}

func (s *Service) CheckAccountsLimit(ctx context.Context, auth0UserId, orgEmail, account string) (bool, error) {
	s.logger.Info("Checking account limit", zap.String("auth0UserId", auth0UserId), zap.String("orgEmail", orgEmail), zap.String("account", account))
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgAccounts, err := s.usageRepo.GetAccountsForOrg(ctx, org[1])
			if err != nil {
				return false, err
			}
			if len(orgAccounts) < int(OrgAccountLimit) {
				return true, nil
			} else if checkAccountInList(account, orgAccounts) {
				return true, nil
			}
			s.logger.Info("Org Account limit reached", zap.String("orgEmail", org[1]))
		}
	}
	userAccounts, err := s.usageRepo.GetAccountsForUser(ctx, auth0UserId)
	if err != nil {
		return false, err
	}
	if len(userAccounts) < int(UserAccountLimit) {
		return true, nil
	} else if checkAccountInList(account, userAccounts) {
		return true, nil
	}
	s.logger.Info("User Account limit reached", zap.String("auth0UserId", auth0UserId))
	return false, nil
}

func checkAccountInList(acc string, accounts []string) bool {
	for _, account := range accounts {
		if acc == account {
			return true
		}
	}
	return false
}
