package statemanager

import (
	"errors"
	"fmt"
	types3 "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/state"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
	"go.uber.org/zap"
	"reflect"
)

func (s *Service) getTransactionByTransactionID(currentState state.State, tid api.TransactionID) transactions.Transaction {
	var transaction transactions.Transaction
	switch tid {
	case api.Transaction_EnsureCredentialExists:
		transaction = transactions.NewEnsureCredentialExists(s.db)
	case api.Transaction_CreateHelmRelease:
		transaction = transactions.NewCreateHelmRelease(s.kubeClient, s.kmsClient, s.cfg, s.db)
	case api.Transaction_CreateInsightBucket:
		transaction = transactions.NewCreateInsightBucket(s.s3Client)
	case api.Transaction_CreateMasterCredential:
		transaction = transactions.NewCreateMasterCredential(s.iamMaster, s.kmsClient, s.cfg, s.db)
	case api.Transaction_CreateOpenSearch:
		transaction = transactions.NewCreateOpenSearch(s.cfg.SecurityGroupID, s.cfg.SubnetID, types3.OpenSearchPartitionInstanceTypeT3SmallSearch, 1, s.db, s.iam, s.opensearch)
	case api.Transaction_CreateIngestionPipeline:
		transaction = transactions.NewCreateIngestionPipeline(s.cfg.SecurityGroupID, s.cfg.SubnetID, s.db, s.osis, s.iam, s.cfg, s.s3Client)
	case api.Transaction_CreateRoleBinding:
		transaction = transactions.NewCreateRoleBinding(s.authClient)
	case api.Transaction_CreateServiceAccountRoles:
		transaction = transactions.NewCreateServiceAccountRoles(s.iam, s.cfg.AWSAccountID, s.cfg.OIDCProvider)
	case api.Transaction_EnsureCredentialOnboarded:
		transaction = transactions.NewEnsureCredentialOnboarded(s.kmsClient, s.cfg, s.db)
	case api.Transaction_EnsureDiscoveryFinished:
		transaction = transactions.NewEnsureDiscoveryFinished(s.cfg)
	case api.Transaction_EnsureJobsFinished:
		transaction = transactions.NewEnsureJobsFinished(s.cfg)
	case api.Transaction_EnsureJobsRunning:
		transaction = transactions.NewEnsureJobsRunning(s.cfg, s.db)
	}
	return transaction
}

func (s *Service) handleTransitionRequirements(workspace *db.Workspace, currentState state.State, currentTransactions []db.WorkspaceTransaction) error {
	allStateTransactionsMet := true
	for _, stateRequirement := range currentState.Requirements(*workspace) {
		alreadyDone := false
		for _, tn := range currentTransactions {
			if tn.TransactionID == stateRequirement {
				alreadyDone = true
			}
		}

		if alreadyDone {
			continue
		}

		transaction := s.getTransactionByTransactionID(currentState, stateRequirement)
		if transaction == nil {
			return fmt.Errorf("failed to find transaction %v", stateRequirement)
		}

		allRequirementsAreMet := true
		for _, transactionRequirement := range transaction.Requirements() {
			found := false
			for _, tn := range currentTransactions {
				if tn.TransactionID == transactionRequirement {
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

		s.logger.Info("applying transaction", zap.String("workspace_id", workspace.ID), zap.String("type", reflect.TypeOf(transaction).String()))
		err := transaction.Apply(*workspace)
		if err != nil {
			if errors.Is(err, transactions.ErrTransactionNeedsTime) {
				return err
			}
			s.logger.Error("failure while applying transaction", zap.String("workspace_id", workspace.ID), zap.String("type", reflect.TypeOf(transaction).String()), zap.Error(err))
			return err
		}

		wt := db.WorkspaceTransaction{WorkspaceID: workspace.ID, TransactionID: stateRequirement}
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
		for _, requirement := range currentState.Requirements(*workspace) {
			if requirement == transactionID.TransactionID {
				isRequirement = true
			}
		}

		if isRequirement {
			continue
		}

		transaction := s.getTransactionByTransactionID(currentState, transactionID.TransactionID)
		if transaction == nil {
			return fmt.Errorf("failed to find transaction %v", transactionID.TransactionID)
		}

		s.logger.Info("rolling back transaction", zap.String("workspace_id", workspace.ID), zap.String("type", reflect.TypeOf(transaction).String()))
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
	for _, v := range state.AllStates(s.db, s.logger) {
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

	err = s.db.UpdateWorkspaceStatus(workspace.ID, currentState.FinishedStateID())
	if err != nil {
		return err
	}

	return nil
}
