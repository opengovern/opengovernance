package grpc_server

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/wastage"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type LimitService struct {
	logger    *zap.Logger
	userRepo  repo.UserRepo
	orgRepo   repo.OrganizationRepo
	usageRepo repo.UsageV2Repo
}

func newLimitService(logger *zap.Logger, userRepo repo.UserRepo, orgRepo repo.OrganizationRepo, usageRepo repo.UsageV2Repo) *LimitService {
	return &LimitService{
		logger:    logger,
		userRepo:  userRepo,
		orgRepo:   orgRepo,
		usageRepo: usageRepo,
	}
}

func (s *LimitService) checkPremiumAndSendErr(userId string, orgEmail string, service string) error {
	user, err := s.userRepo.Get(userId)
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
			org, err := s.orgRepo.Get(orgName[1])
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

func (s *LimitService) CreateUser(echoCtx echo.Context) error {
	var user entity.User
	err := echoCtx.Bind(&user)
	if err != nil {
		return err
	}

	err = s.userRepo.Create(user.ToModel())
	if err != nil {
		return err
	}

	return echoCtx.JSON(http.StatusCreated, user)
}

func (s *LimitService) UpdateUser(echoCtx echo.Context) error {
	idString := echoCtx.Param("userId")
	if idString == "" {
		return errors.New("userId is required")
	}

	premiumUntil, err := strconv.ParseInt(echoCtx.QueryParam("premiumUntil"), 10, 64)
	if err != nil {
		return err
	}

	premiumUntilTime := time.UnixMilli(premiumUntil)
	user := model.User{
		UserId:       idString,
		PremiumUntil: &premiumUntilTime,
	}
	err = s.userRepo.Update(idString, &user)
	if err != nil {
		return err
	}
	return echoCtx.JSON(http.StatusOK, user)
}

func (s *LimitService) CreateOrganization(echoCtx echo.Context) error {
	var org entity.Organization
	err := echoCtx.Bind(&org)
	if err != nil {
		return err
	}

	err = s.orgRepo.Create(org.ToModel())
	if err != nil {
		return err
	}

	return echoCtx.JSON(http.StatusCreated, org)
}

func (s *LimitService) UpdateOrganization(echoCtx echo.Context) error {
	idString := echoCtx.Param("organizationId")
	if idString == "" {
		return errors.New("organizationId is required")
	}

	premiumUntil, err := strconv.ParseInt(echoCtx.QueryParam("premiumUntil"), 10, 64)
	if err != nil {
		return err
	}

	premiumUntilTime := time.UnixMilli(premiumUntil)
	org := model.Organization{
		OrganizationId: idString,
		PremiumUntil:   &premiumUntilTime,
	}
	err = s.orgRepo.Update(idString, &org)
	if err != nil {
		return err
	}
	return echoCtx.JSON(http.StatusOK, org)
}

func (s *LimitService) checkEC2InstanceLimit(auth0UserId, orgEmail string) (bool, error) {
	s.logger.Info("Checking EC2 Instance limit", zap.String("auth0UserId", auth0UserId), zap.String("orgEmail", orgEmail))
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgCount, err := s.usageRepo.GetEC2InstanceOptimizationsCountForOrg(org[1])
			if err != nil {
				return false, err
			}
			if orgCount < int64(wastage.OrgEC2InstanceLimit) {
				return true, nil
			}
			s.logger.Info("Org EC2 Instance limit reached", zap.String("orgEmail", org[1]))
		}
	}
	userCount, err := s.usageRepo.GetEC2InstanceOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(wastage.UserEC2InstanceLimit) {
		return true, nil
	}
	s.logger.Info("User EC2 Instance limit reached", zap.String("auth0UserId", auth0UserId))
	return false, nil
}

func (s *LimitService) checkAccountsLimit(auth0UserId, orgEmail, account string) (bool, error) {
	s.logger.Info("Checking account limit", zap.String("auth0UserId", auth0UserId), zap.String("orgEmail", orgEmail), zap.String("account", account))
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgAccounts, err := s.usageRepo.GetAccountsForOrg(org[1])
			if err != nil {
				return false, err
			}
			if len(orgAccounts) < int(wastage.OrgAccountLimit) {
				return true, nil
			} else if checkAccountInList(account, orgAccounts) {
				return true, nil
			}
			s.logger.Info("Org Account limit reached", zap.String("orgEmail", org[1]))
		}
	}
	userAccounts, err := s.usageRepo.GetAccountsForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if len(userAccounts) < int(wastage.UserAccountLimit) {
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
