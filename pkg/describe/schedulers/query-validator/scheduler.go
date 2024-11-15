package query_runner

import (
	"fmt"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opengovernance/pkg/describe/db/model"
	queryvalidator "github.com/opengovern/opengovernance/pkg/query-validator"
	"go.uber.org/zap"
)

func (s *JobScheduler) runScheduler() error {
	clientCtx := &httpclient.Context{UserRole: api.AdminRole}

	controls, err := s.complianceClient.ListControl(clientCtx, nil, nil)
	if err != nil {
		s.logger.Error("error while listing benchmarks", zap.Error(err))
		return fmt.Errorf("error while listing benchmarks: %v", err)
	}
	for _, c := range controls {
		_, err = s.db.CreateQueryValidatorJob(&model.QueryValidatorJob{
			QueryId:        c.ID,
			QueryType:      queryvalidator.QueryTypeComplianceControl,
			Status:         queryvalidator.QueryValidatorCreated,
			FailureMessage: "",
		})
		if err != nil {
			s.logger.Error("error while creating query-validator job", zap.Error(err))
		}
	}

	namedQueries, err := s.inventoryClient.ListQueriesV2(clientCtx)
	if err != nil {
		s.logger.Error("error while listing benchmarks", zap.Error(err))
		return fmt.Errorf("error while listing benchmarks: %v", err)
	}
	for _, nq := range namedQueries.Items {
		_, err = s.db.CreateQueryValidatorJob(&model.QueryValidatorJob{
			QueryId:        nq.ID,
			QueryType:      queryvalidator.QueryTypeNamedQuery,
			Status:         queryvalidator.QueryValidatorCreated,
			FailureMessage: "",
		})
		if err != nil {
			s.logger.Error("error while creating query-validator job", zap.Error(err))
		}
	}

	return nil
}
