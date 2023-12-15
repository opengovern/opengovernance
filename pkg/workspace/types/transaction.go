package types

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

type Transaction interface {
	Requirements() []TransactionID
	Apply(workspace db.Workspace) error
	Rollback(workspace db.Workspace) error
}

type TransactionID string

const (
	Transaction_CreateServiceAccountRoles    TransactionID = "CreateServiceAccountRoles"
	Transaction_CreateOpenSearch             TransactionID = "CreateOpenSearch"
	Transaction_CreateInsightBucket          TransactionID = "CreateInsightBucket"
	Transaction_CreateRoleBinding            TransactionID = "CreateRoleBinding"
	Transaction_CreateMasterCredential       TransactionID = "CreateMasterCredential"
	Transaction_CreateHelmRelease            TransactionID = "CreateHelmRelease"
	Transaction_EnsureCredentialOnboarded    TransactionID = "EnsureCredentialOnboarded"
	Transaction_EnsureDiscoveryFinished      TransactionID = "EnsureDiscoveryFinished"
	Transaction_EnsureBootstrapInputFinished TransactionID = "EnsureBootstrapInputFinished"
	Transaction_EnsureJobsRunning            TransactionID = "EnsureJobsRunning"
	Transaction_EnsureJobsFinished           TransactionID = "EnsureJobsFinished"
)

var ErrTransactionNeedsTime = errors.New("transaction needs time")
