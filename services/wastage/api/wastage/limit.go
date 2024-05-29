package wastage

import (
	"go.uber.org/zap"
	"strings"
)

func (s API) checkRDSInstanceLimit(auth0UserId, orgEmail string) (bool, error) {
	s.logger.Info("Checking RDS Instance limit", zap.String("auth0UserId", auth0UserId), zap.String("orgEmail", orgEmail))
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgCount, err := s.usageRepo.GetRDSInstanceOptimizationsCountForOrg(org[1])
			if err != nil {
				return false, err
			}
			if orgCount < int64(OrgRDSInstanceLimit) {
				return true, nil
			}
			s.logger.Info("Org RDS Instance limit reached", zap.String("orgEmail", org[1]))
		}
	}
	userCount, err := s.usageRepo.GetRDSInstanceOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserRDSInstanceLimit) {
		return true, nil
	}
	s.logger.Info("User RDS Instance limit reached", zap.String("auth0UserId", auth0UserId))
	return false, nil
}

func (s API) checkRDSClusterLimit(auth0UserId, orgEmail string) (bool, error) {
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgCount, err := s.usageRepo.GetRDSClusterOptimizationsCountForOrg(org[1])
			if err != nil {
				return false, err
			}
			if orgCount < int64(OrgRDSClusterLimit) {
				return true, nil
			}
		}
	}
	userCount, err := s.usageRepo.GetRDSClusterOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserRDSClusterLimit) {
		return true, nil
	}
	return false, nil
}

func (s API) checkEC2InstanceLimit(auth0UserId, orgEmail string) (bool, error) {
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgCount, err := s.usageRepo.GetEC2InstanceOptimizationsCountForOrg(org[1])
			if err != nil {
				return false, err
			}
			if orgCount < int64(OrgEC2InstanceLimit) {
				return true, nil
			}
		}
	}
	userCount, err := s.usageRepo.GetEC2InstanceOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserEC2InstanceLimit) {
		return true, nil
	}
	return false, nil
}

//
//func checkEBSVolumeLimit(db repo.UsageV2Repo, auth0UserId, orgEmail string) (bool, error) {
//	if orgEmail != "" && strings.Contains(orgEmail, "@") {
//		org := strings.Split(orgEmail, "@")
//		if len(org) > 1 {
//			orgCount, err := db.GetEBSVolumeOptimizationsCountForOrg(org[1])
//			if err != nil {
//				return false, err
//			}
//			if orgCount < int64(OrgEBSVolumeLimit) {
//				return true, nil
//			}
//		}
//	}
//	userCount, err := db.GetEBSVolumeOptimizationsCountForUser(auth0UserId)
//	if err != nil {
//		return false, err
//	}
//	if userCount < int64(UserEBSVolumeLimit) {
//		return true, nil
//	}
//	return false, nil
//}

func (s API) checkAccountsLimit(auth0UserId, orgEmail, account string) (bool, error) {
	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgAccounts, err := s.usageRepo.GetAccountsForOrg(org[1])
			if err != nil {
				return false, err
			}
			if len(orgAccounts) < int(OrgAccountLimit) {
				return true, nil
			} else if checkAccountInList(account, orgAccounts) {
				return true, nil
			}
		}
	}
	userAccounts, err := s.usageRepo.GetAccountsForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if len(userAccounts) < int(UserAccountLimit) {
		return true, nil
	} else if checkAccountInList(account, userAccounts) {
		return true, nil
	}
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
