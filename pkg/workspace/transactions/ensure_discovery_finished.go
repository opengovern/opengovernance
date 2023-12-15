package transactions

import (
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
	"strings"
)

type EnsureDiscoveryFinished struct {
	cfg config.Config
}

func NewEnsureDiscoveryFinished(
	cfg config.Config,
) *EnsureDiscoveryFinished {
	return &EnsureDiscoveryFinished{
		cfg: cfg,
	}
}

func (t *EnsureDiscoveryFinished) Requirements() []types.TransactionID {
	return []types.TransactionID{types.Transaction_EnsureBootstrapInputFinished}
}

func (t *EnsureDiscoveryFinished) Apply(workspace db.Workspace) error {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(t.cfg.Scheduler.BaseURL, "%NAMESPACE%", workspace.ID)
	schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)

	status, err := schedulerClient.GetDescribeAllJobsStatus(hctx)
	if err != nil {
		return err
	}

	// waiting for all connections to finish
	if status == nil || *status != api.DescribeAllJobsStatusResourcesPublished {
		return types.ErrTransactionNeedsTime
	}

	return nil
}

func (t *EnsureDiscoveryFinished) Rollback(workspace db.Workspace) error {
	return nil
}
