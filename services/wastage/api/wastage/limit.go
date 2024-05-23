package wastage

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"strings"
)

func checkRDSInstanceLimit(db repo.UsageV2Repo, auth0UserId, orgEmail string) (bool, error) {
	userCount, err := db.GetRDSInstanceOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserRDSInstanceLimit) {
		return true, nil
	}
	org := strings.Split(orgEmail, "@")
	orgCount, err := db.GetRDSInstanceOptimizationsCountForOrg(org[1])
	if err != nil {
		return false, err
	}
	if orgCount < int64(OrgRDSInstanceLimit) {
		return true, nil
	}
	return false, nil
}

func checkRDSClusterLimit(db repo.UsageV2Repo, auth0UserId, orgEmail string) (bool, error) {
	userCount, err := db.GetRDSClusterOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserRDSClusterLimit) {
		return true, nil
	}
	org := strings.Split(orgEmail, "@")
	orgCount, err := db.GetRDSClusterOptimizationsCountForOrg(org[1])
	if err != nil {
		return false, err
	}
	if orgCount < int64(OrgRDSClusterLimit) {
		return true, nil
	}
	return false, nil
}

func checkEC2InstanceLimit(db repo.UsageV2Repo, auth0UserId, orgEmail string) (bool, error) {
	userCount, err := db.GetEC2InstanceOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserEC2InstanceLimit) {
		return true, nil
	}
	org := strings.Split(orgEmail, "@")
	orgCount, err := db.GetEC2InstanceOptimizationsCountForOrg(org[1])
	if err != nil {
		return false, err
	}
	if orgCount < int64(OrgEC2InstanceLimit) {
		return true, nil
	}
	return false, nil
}

func checkEBSVolumeLimit(db repo.UsageV2Repo, auth0UserId, orgEmail string) (bool, error) {
	userCount, err := db.GetEBSVolumeOptimizationsCountForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if userCount < int64(UserEBSVolumeLimit) {
		return true, nil
	}
	org := strings.Split(orgEmail, "@")
	orgCount, err := db.GetEBSVolumeOptimizationsCountForOrg(org[1])
	if err != nil {
		return false, err
	}
	if orgCount < int64(OrgEBSVolumeLimit) {
		return true, nil
	}
	return false, nil
}

func checkAccountsLimit(db repo.UsageV2Repo, auth0UserId, orgEmail, account string) (bool, error) {
	userAccounts, err := db.GetAccountsForUser(auth0UserId)
	if err != nil {
		return false, err
	}
	if len(userAccounts) < int(UserAccountLimit) {
		return true, nil
	} else if checkAccountInList(account, userAccounts) {
		return true, nil
	}
	org := strings.Split(orgEmail, "@")
	orgAccounts, err := db.GetAccountsForOrg(org[1])
	if err != nil {
		return false, err
	}
	if len(orgAccounts) < int(OrgAccountLimit) {
		return true, nil
	} else if checkAccountInList(account, orgAccounts) {
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
