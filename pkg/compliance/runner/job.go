package runner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"time"
)

type Caller struct {
	RootBenchmark      string
	ParentBenchmarkIDs []string
	PolicyID           string
	PolicySeverity     types.FindingSeverity
}

type ExecutionPlan struct {
	Callers        []Caller
	QueryID        string
	QueryEngine    string
	QueryConnector source.Type

	ConnectionID         *string
	ResourceCollectionID *string
}

type Job struct {
	ID        uint
	CreatedAt time.Time

	ExecutionPlan ExecutionPlan
}

type JobConfig struct {
	config           Config
	logger           *zap.Logger
	complianceClient complianceClient.ComplianceServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	inventoryClient  inventoryClient.InventoryServiceClient
	steampipeConn    *steampipe.Database
	esClient         kaytu.Client
	kafkaProducer    *confluent_kafka.Producer
}

func (j *Job) Initialize(ctx context.Context, jc JobConfig) error {
	providerAccountID := "all"
	if j.ExecutionPlan.ConnectionID != nil &&
		*j.ExecutionPlan.ConnectionID != "" &&
		*j.ExecutionPlan.ConnectionID != "all" {
		conn, err := jc.onboardClient.GetSource(&httpclient.Context{UserRole: api.InternalRole}, *j.ExecutionPlan.ConnectionID)
		if err != nil {
			jc.logger.Error("failed to get source", zap.Error(err), zap.String("connection_id", *j.ExecutionPlan.ConnectionID))
			return err
		}
		providerAccountID = conn.ConnectionID
	}

	err := jc.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID, providerAccountID)
	if err != nil {
		jc.logger.Error("failed to set account id", zap.Error(err))
		return err
	}
	err = jc.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType, "compliance")
	if err != nil {
		jc.logger.Error("failed to set client type", zap.Error(err))
		return err
	}

	if j.ExecutionPlan.ResourceCollectionID != nil {
		rc, err := jc.inventoryClient.GetResourceCollection(&httpclient.Context{UserRole: api.InternalRole}, *j.ExecutionPlan.ResourceCollectionID)
		if err != nil {
			jc.logger.Error("failed to get resource collection", zap.Error(err), zap.String("rc_id", *j.ExecutionPlan.ResourceCollectionID))
			return err
		}
		filtersJson, err := json.Marshal(rc.Filters)
		if err != nil {
			jc.logger.Error("failed to marshal resource collection filters", zap.Error(err), zap.String("rc_id", *j.ExecutionPlan.ResourceCollectionID))
			return err
		}
		err = jc.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyResourceCollectionFilters, base64.StdEncoding.EncodeToString(filtersJson))
		if err != nil {
			jc.logger.Error("failed to set resource collection filters", zap.Error(err), zap.String("rc_id", *j.ExecutionPlan.ResourceCollectionID))
			return err
		}
	}

	return nil
}

func (j *Job) Run(jc JobConfig) error {
	ctx := context.Background()

	jc.logger.Info("Running query",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.QueryID),
		zap.Stringp("query_id", j.ExecutionPlan.ConnectionID),
		zap.Stringp("rc_id", j.ExecutionPlan.ResourceCollectionID),
	)

	if err := j.Initialize(ctx, jc); err != nil {
		return err
	}
	defer jc.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID)
	defer jc.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType)
	defer jc.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyResourceCollectionFilters)

	query, err := jc.complianceClient.GetQuery(&httpclient.Context{UserRole: api.InternalRole}, j.ExecutionPlan.QueryID)
	if err != nil {
		return err
	}

	res, err := jc.steampipeConn.QueryAll(ctx, query.QueryToExecute)
	if err != nil {
		return err
	}

	jc.logger.Info("Extracting and pushing to kafka",
		zap.Uint("job_id", j.ID),
		zap.Int("res_count", len(res.Data)),
		zap.Int("caller_count", len(j.ExecutionPlan.Callers)),
	)

	for _, caller := range j.ExecutionPlan.Callers {
		findings, err := j.ExtractFindings(jc.logger, caller, res, *query)
		if err != nil {
			return err
		}

		var docs []kafka.Doc
		for _, f := range findings {
			docs = append(docs, f)
		}

		err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, docs, jc.logger, nil)
		if err != nil {
			jc.logger.Error("failed to send findings", zap.Error(err), zap.String("benchmark_id", caller.RootBenchmark), zap.String("policy_id", caller.PolicyID))
			return err
		}

		err = RemoveOldFindings(jc, j.ID, *j.ExecutionPlan.ConnectionID, j.ExecutionPlan.ResourceCollectionID, caller.RootBenchmark, caller.PolicyID)
		if err != nil {
			jc.logger.Error("failed to remove old findings", zap.Error(err), zap.String("benchmark_id", caller.RootBenchmark), zap.String("policy_id", caller.PolicyID))
			return err
		}
	}
	jc.logger.Info("Finished job",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.QueryID),
		zap.Stringp("query_id", j.ExecutionPlan.ConnectionID),
		zap.Stringp("rc_id", j.ExecutionPlan.ResourceCollectionID),
	)
	return nil
}

type DocIdResponse struct {
	Took int `json:"took"`
	Hits struct {
		Total struct {
			Value int `json:"value"`
		}
		Hits []struct {
			ID   string   `json:"_id"`
			Sort []string `json:"sort"`
		}
	}
}

func RemoveOldFindings(jc JobConfig, jobID uint,
	connectionId string,
	resourceCollectionId *string,
	benchmarkID,
	policyID string) error {
	ctx := context.Background()
	idx := types.FindingsIndex
	if resourceCollectionId != nil {
		idx = types.ResourceCollectionsFindingsIndex
	}
	var filters []map[string]any
	mustFilters := make([]map[string]any, 0, 4)
	mustFilters = append(mustFilters, map[string]any{
		"term": map[string]any{
			"benchmarkID": benchmarkID,
		},
	})
	mustFilters = append(mustFilters, map[string]any{
		"term": map[string]any{
			"policyID": policyID,
		},
	})
	mustFilters = append(mustFilters, map[string]any{
		"term": map[string]any{
			"connectionID": connectionId,
		},
	})
	if resourceCollectionId != nil {
		mustFilters = append(mustFilters, map[string]any{
			"term": map[string]any{
				"resourceCollection": resourceCollectionId,
			},
		})
	}

	filters = append(filters, map[string]any{
		"bool": map[string]any{
			"must": []map[string]any{
				{
					"bool": map[string]any{
						"must_not": map[string]any{
							"term": map[string]any{
								"complianceJobID": jobID,
							},
						},
					},
				},
				{
					"bool": map[string]any{
						"filter": mustFilters,
					},
				},
			},
		},
	})

	request := make(map[string]any)
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	request["size"] = 10000
	request["_source"] = false
	request["sort"] = map[string]any{
		"_id": "asc",
	}
	searchAfter := make([]string, 0)

	for {
		request["search_after"] = searchAfter
		query, err := json.Marshal(request)
		if err != nil {
			return err
		}

		res := DocIdResponse{}
		err = jc.esClient.Search(ctx, idx, string(query), &res)
		if err != nil {
			jc.logger.Error("failed to search", zap.Error(err), zap.String("query", string(query)), zap.String("index", idx))
			return err
		}
		if len(res.Hits.Hits) == 0 {
			break
		}
		searchAfter = res.Hits.Hits[len(res.Hits.Hits)-1].Sort

		tombstoneDocs := make([]*confluent_kafka.Message, 0, 10000)
		for _, hit := range res.Hits.Hits {
			msg := kafka.Msg(hit.ID, nil, idx, jc.config.Kafka.Topic, confluent_kafka.PartitionAny)
			tombstoneDocs = append(tombstoneDocs, msg)
		}
		err = kafka.SyncSendWithRetry(jc.logger, jc.kafkaProducer, tombstoneDocs, nil, 5)
		if err != nil {
			jc.logger.Error("failed to send tombstone docs", zap.Error(err), zap.String("index", idx))
			return err
		}
	}

	return nil
}
