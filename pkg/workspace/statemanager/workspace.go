package statemanager

import (
	"errors"
	"fmt"
	types3 "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/state"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
	"go.uber.org/zap"
	"reflect"
)

func (s *Service) getTransactionByTransactionID(tid transactions.TransactionID) transactions.Transaction {
	var transaction transactions.Transaction
	switch tid {
	case transactions.Transaction_CreateHelmRelease:
		transaction = transactions.NewCreateHelmRelease(s.kubeClient, s.kmsClient, s.cfg, s.db)
	case transactions.Transaction_CreateInsightBucket:
		transaction = transactions.NewCreateInsightBucket(s.s3Client)
	case transactions.Transaction_CreateMasterCredential:
		transaction = transactions.NewCreateMasterCredential(s.iamMaster, s.kmsClient, s.cfg, s.db)
	case transactions.Transaction_CreateOpenSearch:
		transaction = transactions.NewCreateOpenSearch(s.cfg.MasterRoleARN, s.cfg.SecurityGroupID, s.cfg.SubnetID, types3.OpenSearchPartitionInstanceTypeT3SmallSearch, 1, s.db, s.opensearch)
	case transactions.Transaction_CreateRoleBinding:
		transaction = transactions.NewCreateRoleBinding(s.authClient)
	case transactions.Transaction_CreateServiceAccountRoles:
		transaction = transactions.NewCreateServiceAccountRoles(s.iam, s.cfg.AWSAccountID, s.cfg.OIDCProvider)
	case transactions.Transaction_EnsureBootstrapInputFinished:
		transaction = transactions.NewEnsureBootstrapInputFinished()
	case transactions.Transaction_EnsureCredentialOnboarded:
		transaction = transactions.NewEnsureCredentialOnboarded(s.kmsClient, s.cfg, s.db)
	case transactions.Transaction_EnsureDiscoveryFinished:
		transaction = transactions.NewEnsureDiscoveryFinished(s.cfg)
	case transactions.Transaction_EnsureJobsFinished:
		transaction = transactions.NewEnsureJobsFinished(s.cfg)
	case transactions.Transaction_EnsureJobsRunning:
		transaction = transactions.NewEnsureJobsRunning(s.cfg, s.db)
	}
	return transaction
}

func (s *Service) handleTransitionRequirements(workspace *db.Workspace, currentState state.State, currentTransactions []db.WorkspaceTransaction) error {
	allStateTransactionsMet := true
	for _, stateRequirement := range currentState.Requirements() {
		alreadyDone := false
		for _, tn := range currentTransactions {
			if transactions.TransactionID(tn.TransactionID) == stateRequirement {
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
				if transactions.TransactionID(tn.TransactionID) == transactionRequirement {
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

		s.logger.Info("applying transaction", zap.String("type", reflect.TypeOf(transaction).String()))
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
		return transactions.ErrTransactionNeedsTime
	}
	return nil
}

func (s *Service) handleTransitionRollbacks(workspace *db.Workspace, currentState state.State, currentTransactions []db.WorkspaceTransaction) error {
	for _, transactionID := range currentTransactions {
		isRequirement := false
		for _, requirement := range currentState.Requirements() {
			if requirement == transactions.TransactionID(transactionID.TransactionID) {
				isRequirement = true
			}
		}

		if isRequirement {
			continue
		}

		transaction := s.getTransactionByTransactionID(transactions.TransactionID(transactionID.TransactionID))
		if transaction == nil {
			return fmt.Errorf("failed to find transaction %v", transactionID.TransactionID)
		}

		s.logger.Info("rolling back transaction", zap.String("type", reflect.TypeOf(transaction).String()))
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
	var currentState state.State
	for _, v := range state.AllStates {
		if v.ProcessingStateID() == state.StateID(workspace.Status) {
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
		if errors.Is(err, transactions.ErrTransactionNeedsTime) {
			return nil
		}
		return err
	}

	err = s.handleTransitionRollbacks(workspace, currentState, tns)
	if err != nil {
		if errors.Is(err, transactions.ErrTransactionNeedsTime) {
			return nil
		}
		return err
	}

	err = s.db.UpdateWorkspaceStatus(workspace.ID, string(currentState.FinishedStateID()))
	if err != nil {
		return err
	}

	return nil
}
