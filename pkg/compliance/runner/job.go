package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"io"
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
	complianceClient client.ComplianceServiceClient
	onboardClient    client2.OnboardServiceClient
	steampipeConn    *steampipe.Database
	esClient         kaytu.Client
	kafkaProducer    *confluent_kafka.Producer
}

func (j *Job) Initialize(ctx context.Context, jc JobConfig) error {
	providerAccountID := "all"
	if j.ExecutionPlan.ConnectionID != nil {
		conn, err := jc.onboardClient.GetSource(&httpclient.Context{UserRole: api.InternalRole}, *j.ExecutionPlan.ConnectionID)
		if err != nil {
			return err
		}
		providerAccountID = conn.ConnectionID
	}

	err := jc.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID, providerAccountID)
	if err != nil {
		return err
	}
	defer jc.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID)
	err = jc.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType, "compliance")
	if err != nil {
		return err
	}
	defer jc.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType)
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
		findings, err := j.ExtractFindings(caller, res, jc)
		if err != nil {
			return err
		}

		var docs []kafka.Doc
		for _, f := range findings {
			docs = append(docs, f)
		}

		err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, docs, jc.logger, nil)
		if err != nil {
			return err
		}

		//TODO-Saleh fix this
		//err = j.RemoveOldFindings(jc)
		//if err != nil {
		//	return err
		//}
	}
	jc.logger.Info("Finished job",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.QueryID),
		zap.Stringp("query_id", j.ExecutionPlan.ConnectionID),
		zap.Stringp("rc_id", j.ExecutionPlan.ResourceCollectionID),
	)
	return nil
}

func RemoveOldFindings(jc JobConfig, jobID uint, benchmarkID string) error {
	ctx := context.Background()
	es := jc.esClient.ES()

	index := []string{types.FindingsIndex, types.ResourceCollectionsFindingsIndex}

	var filters []map[string]any
	filters = append(filters, map[string]any{
		"bool": map[string]any{
			"should": []map[string]any{
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
						"must": map[string]any{
							"term": map[string]any{
								"parentBenchmarks": benchmarkID,
							},
						},
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

	query, err := json.Marshal(request)
	if err != nil {
		return err
	}

	jc.logger.Info("delete by query", zap.String("body", string(query)))

	res, err := es.DeleteByQuery(
		index,
		bytes.NewReader(query),
		es.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		return err
	}

	defer kaytu.CloseSafe(res)
	if err != nil {
		b, _ := io.ReadAll(res.Body)
		fmt.Printf("failure while deleting es: %v\n%s\n", err, string(b))
		return err
	} else if err := kaytu.CheckError(res); err != nil {
		if kaytu.IsIndexNotFoundErr(err) {
			return nil
		}
		b, _ := io.ReadAll(res.Body)
		fmt.Printf("failure while querying es: %v\n%s\n", err, string(b))
		return err
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	return nil
}
