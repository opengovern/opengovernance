package worker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"encoding/base64"

	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"k8s.io/apimachinery/pkg/runtime"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	"github.com/kaytu-io/kaytu-engine/pkg/config"
	client3 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	apiOnboard "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	kuberTypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	client4 "sigs.k8s.io/controller-runtime/pkg/client"
)

type Job struct {
	JobID         uint
	ScheduleJobID uint
	DescribedAt   int64
	EvaluatedAt   int64

	ConnectionID string
	BenchmarkID  string

	ConfigReg string
	Connector source.Type
	IsStack   bool
}

type JobResult struct {
	JobID           uint
	Status          api.ComplianceReportJobStatus
	ReportCreatedAt int64
	Error           string
}

func (j *Job) Do(
	complianceClient client.ComplianceServiceClient,
	onboardClient client2.OnboardServiceClient,
	scheduleClient client3.SchedulerServiceClient,
	elasticSearchConfig config.ElasticSearch,
	kfkProducer *confluent_kafka.Producer,
	kfkTopic string,
	currentWorkspaceId string,
	logger *zap.Logger,
) JobResult {
	result := JobResult{
		JobID:           j.JobID,
		Status:          api.ComplianceReportJobCompleted,
		ReportCreatedAt: time.Now().UnixMilli(),
		Error:           "",
	}

	if err := j.Run(complianceClient, onboardClient, scheduleClient, elasticSearchConfig, kfkProducer, kfkTopic, currentWorkspaceId, logger); err != nil {
		result.Error = err.Error()
		result.Status = api.ComplianceReportJobCompletedWithFailure
	}
	result.ReportCreatedAt = time.Now().UnixMilli()
	return result
}

func (j *Job) RunBenchmark(esk keibi.Client, benchmarkID string, complianceClient client.ComplianceServiceClient, steampipeConn *steampipe.Database, connector source.Type) ([]types.Finding, error) {
	ctx := &httpclient.Context{
		UserRole: api2.AdminRole,
	}

	benchmark, err := complianceClient.GetBenchmark(ctx, benchmarkID)
	if err != nil {
		return nil, err
	}

	fmt.Println("+++++++++ Running Benchmark:", benchmarkID)

	var findings []types.Finding
	for _, childBenchmarkID := range benchmark.Children {
		f, err := j.RunBenchmark(esk, childBenchmarkID, complianceClient, steampipeConn, connector)
		if err != nil {
			return nil, err
		}

		findings = append(findings, f...)
	}

	for _, policyID := range benchmark.Policies {
		fmt.Println("+++++++++++ Running Policy:", policyID)
		policy, err := complianceClient.GetPolicy(ctx, policyID)
		if err != nil {
			return nil, err
		}
		if policy.QueryID == nil {
			continue
		}

		query, err := complianceClient.GetQuery(ctx, *policy.QueryID)
		if err != nil {
			return nil, err
		}

		if query.Connector != string(connector) {
			return nil, errors.New("connector doesn't match")
		}

		res, err := steampipeConn.QueryAll(query.QueryToExecute)
		if err != nil {
			return nil, err
		}

		f, err := j.ExtractFindings(esk, benchmark, policy, query, res)
		if err != nil {
			return nil, err
		}

		if !j.IsStack {
			findingsFiltered, err := j.FilterFindings(esk, policyID, f)
			if err != nil {
				return nil, err
			}
			fmt.Println("++++++ findingsFiltered len: ", len(findingsFiltered))
			f = findingsFiltered
		}

		findings = append(findings, f...)
	}

	return findings, nil
}

func (j *Job) Run(complianceClient client.ComplianceServiceClient, onboardClient client2.OnboardServiceClient, schedulerClient client3.SchedulerServiceClient,
	elasticSearchConfig config.ElasticSearch, kfkProducer *confluent_kafka.Producer, kfkTopic string, currentWorkspaceId string, logger *zap.Logger) error {

	ctx := &httpclient.Context{
		UserRole: api2.AdminRole,
	}
	var accountId string
	var connector source.Type
	var esk keibi.Client
	if j.IsStack == true {
		stack, err := schedulerClient.GetStack(ctx, j.ConnectionID)
		if err != nil {
			return err
		}
		accountId = stack.AccountIDs[0]
		connector = stack.SourceType

		eskConfig, err := getStackElasticConfig(currentWorkspaceId, stack.StackID)
		esk, err = keibi.NewClient(keibi.ClientConfig{
			Addresses: []string{eskConfig.Address},
			Username:  &eskConfig.Username,
			Password:  &eskConfig.Password,
			AccountID: &accountId,
		})
		if err != nil {
			return err
		}
	} else {
		src, err := onboardClient.GetSource(ctx, j.ConnectionID)
		if err != nil {
			return err
		}
		accountId = src.ConnectionID
		connector = src.Connector
		if src.LifecycleState != apiOnboard.ConnectionLifecycleStateOnboard {
			return errors.New("connection not healthy")
		}

		esk, err = keibi.NewClient(keibi.ClientConfig{
			Addresses: []string{elasticSearchConfig.Address},
			Username:  &elasticSearchConfig.Username,
			Password:  &elasticSearchConfig.Password,
			AccountID: &accountId,
		})
		if err != nil {
			return err
		}
	}

	fmt.Println("+++++ New elasticSearch Client created")

	err := j.PopulateSteampipeConfig(elasticSearchConfig, accountId)
	if err != nil {
		return err
	}

	cmd := exec.Command("steampipe", "plugin", "list")
	cmdOut, err := cmd.Output()
	if err != nil {
		logger.Error("plugin list failed", zap.Error(err), zap.String("body", string(cmdOut)))
		return err
	}

	cmd = exec.Command("steampipe", "service", "stop", "--force")
	err = cmd.Start()
	if err != nil {
		logger.Error("first stop failed", zap.Error(err))
		return err
	}
	time.Sleep(5 * time.Second)
	//NOTE: stop must be called twice. it's not a mistake
	cmd = exec.Command("steampipe", "service", "stop", "--force")
	err = cmd.Start()
	if err != nil {
		logger.Error("second stop failed", zap.Error(err))
		return err
	}
	time.Sleep(5 * time.Second)

	cmd = exec.Command("steampipe", "service", "start", "--database-listen", "network", "--database-port",
		"9193", "--database-password", "abcd")
	cmdOut, err = cmd.Output()
	if err != nil {
		logger.Error("start failed", zap.Error(err), zap.String("body", string(cmdOut)))
		return err
	}
	time.Sleep(5 * time.Second)

	fmt.Println("+++++ Steampipe service started")
	steampipeConn, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: "localhost",
		Port: "9193",
		User: "steampipe",
		Pass: "abcd",
		Db:   "steampipe",
	})
	if err != nil {
		return err
	}

	fmt.Println("+++++ Steampipe database created")

	findings, err := j.RunBenchmark(esk, j.BenchmarkID, complianceClient, steampipeConn, connector)
	if err != nil {
		return err
	}
	fmt.Println("++++++ findings len: ", len(findings))
	var docs []kafka.Doc
	for _, finding := range findings {
		docs = append(docs, finding)
	}
	return kafka.DoSend(kfkProducer, kfkTopic, -1, docs, logger)
}

func (j *Job) FilterFindings(esClient keibi.Client, policyID string, findings []types.Finding) ([]types.Finding, error) {
	// get all active findings from ES page by page
	// go through the ones extracted and remove duplicates
	// if a finding fetched from es is not duplicated disable it
	from := 0
	const esFetchSize = 1000
	for {
		resp, err := es.GetActiveFindings(esClient, policyID, from, esFetchSize)
		if err != nil {
			return nil, err
		}
		fmt.Println("+++++++++ active old findings:", len(resp.Hits.Hits))
		from += esFetchSize

		if len(resp.Hits.Hits) == 0 {
			break
		}

		for _, hit := range resp.Hits.Hits {
			dup := false

			for idx, finding := range findings {
				if finding.ResourceID == hit.Source.ResourceID && finding.PolicyID == hit.Source.PolicyID {
					dup = true
					fmt.Println("+++++++++ removing dup:", finding.ID, hit.Source.ID)
					findings = append(findings[:idx], findings[idx+1:]...)
					break
				}
			}

			if !dup {
				f := hit.Source
				f.StateActive = false
				fmt.Println("+++++++++ making this disabled:", f.ID)
				findings = append(findings, f)
			}
		}
	}
	return findings, nil
}

func (j *Job) ExtractFindings(client keibi.Client, benchmark *api.Benchmark, policy *api.Policy, query *api.Query, res *steampipe.Result) ([]types.Finding, error) {
	var findings []types.Finding
	resourceType := ""
	for _, record := range res.Data {
		if len(record) != len(res.Headers) {
			return nil, fmt.Errorf("invalid record length, record=%d headers=%d", len(record), len(res.Headers))
		}
		recordValue := make(map[string]any)
		for idx, header := range res.Headers {
			value := record[idx]
			recordValue[header] = value
		}

		var resourceID, resourceName, resourceLocation, reason string
		var status types.ComplianceResult
		if v, ok := recordValue["resource"].(string); ok {
			resourceID = v
			if resourceType == "" {
				lookupResource, err := es.FetchLookupsByResourceIDWildcard(client, resourceID)
				if err != nil {
					return nil, err
				}
				if lookupResource.Hits.Total.Value > 0 {
					resourceType = lookupResource.Hits.Hits[0].Source.ResourceType
				}
			}
		}
		if v, ok := recordValue["name"].(string); ok {
			resourceName = v
		}
		if v, ok := recordValue["location"].(string); ok {
			resourceLocation = v
		}
		if v, ok := recordValue["reason"].(string); ok {
			reason = v
		}
		if v, ok := recordValue["status"].(string); ok {
			status = types.ComplianceResult(v)
		}
		fmt.Println("======", recordValue)

		severity := types.FindingSeverityNone
		if status == types.ComplianceResultALARM {
			severity = policy.Severity
		}
		findings = append(findings, types.Finding{
			ID:               fmt.Sprintf("%s-%s-%d", resourceID, policy.ID, j.ScheduleJobID),
			BenchmarkID:      benchmark.ID,
			PolicyID:         policy.ID,
			ConnectionID:     j.ConnectionID,
			DescribedAt:      j.DescribedAt,
			EvaluatedAt:      j.EvaluatedAt,
			StateActive:      true,
			Result:           status,
			Severity:         severity,
			Evaluator:        query.Engine,
			Connector:        j.Connector,
			ResourceID:       resourceID,
			ResourceName:     resourceName,
			ResourceLocation: resourceLocation,
			ResourceType:     resourceType,
			Reason:           reason,
			ComplianceJobID:  j.JobID,
			ScheduleJobID:    j.ScheduleJobID,
		})
	}
	return findings, nil
}

func getStackElasticConfig(workspaceId string, stackId string) (config.ElasticSearch, error) {

	scheme := runtime.NewScheme()
	if err := helmv2.AddToScheme(scheme); err != nil {
		return config.ElasticSearch{}, err
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return config.ElasticSearch{}, err
	}
	kubeClient, err := client4.New(ctrl.GetConfigOrDie(), client4.Options{Scheme: scheme})
	if err != nil {
		return config.ElasticSearch{}, err
	}

	releaseName := stackId
	secretName := fmt.Sprintf("%s-es-elastic-user", releaseName)

	secret := &corev1.Secret{}
	err = kubeClient.Get(context.TODO(), kuberTypes.NamespacedName{
		Namespace: workspaceId,
		Name:      secretName,
	}, secret)
	if err != nil {
		return config.ElasticSearch{}, err
	}
	password, err := base64.URLEncoding.DecodeString(string(secret.Data["elastic"]))
	if err != nil {
		return config.ElasticSearch{}, err
	}
	return config.ElasticSearch{
		Address:  fmt.Sprintf("https://%s-es-http:9200/", releaseName),
		Username: "elastic",
		Password: string(password),
	}, nil
}
