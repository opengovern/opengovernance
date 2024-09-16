package service

import (
	"context"
	shared_entities "github.com/kaytu-io/kaytu-util/pkg/api/shared-entities"
	"github.com/kaytu-io/open-governance/services/information/config"
	"github.com/kaytu-io/open-governance/services/information/db/model"
	"github.com/kaytu-io/open-governance/services/information/db/repo"
	"go.uber.org/zap"
)

type InformationService struct {
	cfg    config.InformationConfig
	logger *zap.Logger

	csmpUsageRepo repo.CspmUsageRepo
}

func NewInformationService(cfg config.InformationConfig, logger *zap.Logger, cspmUsageRepo repo.CspmUsageRepo) *InformationService {
	return &InformationService{
		cfg:           cfg,
		logger:        logger.Named("information-service"),
		csmpUsageRepo: cspmUsageRepo,
	}
}

func (s *InformationService) RecordUsage(ctx context.Context, req shared_entities.CspmUsageRequest) error {

	m := model.CspmUsage{
		WorkspaceId:            req.WorkspaceId,
		GatherTimestamp:        req.GatherTimestamp,
		Hostname:               req.Hostname,
		AwsAccountCount:        req.AwsAccountCount,
		AzureSubscriptionCount: req.AzureSubscriptionCount,
		ApproximateSpend:       req.ApproximateSpend,
	}

	if err := s.csmpUsageRepo.Create(ctx, &m); err != nil {
		s.logger.Error("failed to create cspm usage", zap.Error(err))
		return err
	}
	return nil
}
