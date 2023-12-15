package statemanager

import (
	"errors"
	"fmt"
	types3 "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
)

func (s *Service) getTransactionByTransactionID(tid types.TransactionID) types.Transaction {
	var transaction types.Transaction
	switch tid {
	case types.Transaction_CreateHelmRelease:
		transaction = transactions.NewCreateHelmRelease(s.kubeClient, s.kmsClient, s.cfg, s.db)
	case types.Transaction_CreateInsightBucket:
		transaction = transactions.NewCreateInsightBucket(s.s3Client)
	case types.Transaction_CreateMasterCredential:
		transaction = transactions.NewCreateMasterCredential(s.iamMaster, s.kmsClient, s.cfg, s.db)
	case types.Transaction_CreateOpenSearch:
		transaction = transactions.NewCreateOpenSearch(s.cfg.MasterRoleARN, s.cfg.SecurityGroupID, s.cfg.SubnetID, types3.OpenSearchPartitionInstanceTypeT3SmallSearch, 1, s.db, s.opensearch)
	case types.Transaction_CreateRoleBinding:
		transaction = transactions.NewCreateRoleBinding(s.authClient)
	case types.Transaction_CreateServiceAccountRoles:
		transaction = transactions.NewCreateServiceAccountRoles(s.iam, s.cfg.AWSAccountID, s.cfg.OIDCProvider)
	case types.Transaction_EnsureBootstrapInputFinished:
		transaction = transactions.NewEnsureBootstrapInputFinished()
	case types.Transaction_EnsureCredentialOnboarded:
		transaction = transactions.NewEnsureCredentialOnboarded(s.kmsClient, s.cfg, s.db)
	case types.Transaction_EnsureDiscoveryFinished:
		transaction = transactions.NewEnsureDiscoveryFinished(s.cfg)
	case types.Transaction_EnsureJobsFinished:
		transaction = transactions.NewEnsureJobsFinished(s.cfg)
	case types.Transaction_EnsureJobsRunning:
		transaction = transactions.NewEnsureJobsRunning(s.cfg, s.db)
	}
	return transaction
}

func (s *Service) handleTransitionRequirements(workspace *db.Workspace, currentState types.State, currentTransactions []db.WorkspaceTransaction) error {
	allStateTransactionsMet := true
	for _, stateRequirement := range currentState.Requirements() {
		alreadyDone := false
		for _, tn := range currentTransactions {
			if types.TransactionID(tn.TransactionID) == stateRequirement {
				alreadyDone = true
			}
		}

		if alreadyDone {
			continue
		}

		transaction := s.getTransactionByTransactionID(stateRequirement)
		if transaction == nil {
			return fmt.Errorf("failed to find transaction %v", stateRequirement)
		}

		allRequirementsAreMet := true
		for _, transactionRequirement := range transaction.Requirements() {
			found := false
			for _, tn := range currentTransactions {
				if types.TransactionID(tn.TransactionID) == transactionRequirement {
					found = true
				}
			}

			if !found {
				allRequirementsAreMet = false
			}
		}

		if !allRequirementsAreMet {
			allStateTransactionsMet = false
			continue
		}

		err := transaction.Apply(*workspace)
		if err != nil {
			return err
		}

		wt := db.WorkspaceTransaction{WorkspaceID: workspace.ID, TransactionID: string(stateRequirement)}
		err = s.db.CreateWorkspaceTransaction(&wt)
		if err != nil {
			return err
		}
	}

	if !allStateTransactionsMet {
		return types.ErrTransactionNeedsTime
	}
	return nil
}

func (s *Service) handleTransitionRollbacks(workspace *db.Workspace, currentState types.State, currentTransactions []db.WorkspaceTransaction) error {
	for _, transactionID := range currentTransactions {
		isRequirement := false
		for _, requirement := range currentState.Requirements() {
			if requirement == types.TransactionID(transactionID.TransactionID) {
				isRequirement = true
			}
		}

		if isRequirement {
			continue
		}

		transaction := s.getTransactionByTransactionID(types.TransactionID(transactionID.TransactionID))
		if transaction == nil {
			return fmt.Errorf("failed to find transaction %v", transactionID.TransactionID)
		}

		err := transaction.Rollback(*workspace)
		if err != nil {
			return err
		}

		err = s.db.DeleteWorkspaceTransaction(workspace.ID, transactionID.TransactionID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) handleTransition(workspace *db.Workspace) error {
	var currentState types.State
	for _, v := range types.AllStates {
		if v.ProcessingStateID() == workspace.Status {
			currentState = v
		}
	}

	if currentState == nil {
		// no transition
		return nil
	}

	tns, err := s.db.GetTransactionsByWorkspace(workspace.ID)
	if err != nil {
		return err
	}

	err = s.handleTransitionRequirements(workspace, currentState, tns)
	if err != nil {
		if errors.Is(err, types.ErrTransactionNeedsTime) {
			return nil
		}
		return err
	}

	err = s.handleTransitionRollbacks(workspace, currentState, tns)
	if err != nil {
		if errors.Is(err, types.ErrTransactionNeedsTime) {
			return nil
		}
		return err
	}

	err = s.db.UpdateWorkspaceStatus(workspace.ID, currentState.FinishedStateID())
	if err != nil {
		return err
	}

	return nil
}
