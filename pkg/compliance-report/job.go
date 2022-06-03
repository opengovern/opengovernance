package compliance_report

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	aws "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	azure "gitlab.com/keibiengine/keibi-engine/pkg/azure/model"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

var DoComplianceReportJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "compliance_worker",
	Name:      "do_compliance_report_jobs_total",
	Help:      "Count of done compliance report jobs in compliance-worker service",
}, []string{"provider", "status"})

var DoComplianceReportJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "compliance_worker",
	Name:      "do_compliance_report_jobs_duration_seconds",
	Help:      "Duration of done compliance report jobs in compliance-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"provider", "status"})

var DoComplianceReportCleanupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "compliance_cleanup_worker",
	Name:      "do_compliance_report_cleanup_jobs_total",
	Help:      "Count of done compliance report cleanup jobs in compliance-cleanup-worker service",
}, []string{"provider", "status"})

var DoComplianceReportCleanupJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "compliance_cleanup_worker",
	Name:      "do_compliance_report_cleanup_jobs_duration_seconds",
	Help:      "Duration of done compliance report cleanup jobs in compliance-cleanup-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"provider", "status"})

const (
	cleanupJobTimeout = 5 * time.Minute
)

type Job struct {
	ReportID    uint
	JobID       uint
	SourceID    uuid.UUID
	SourceType  source.Type
	ConfigReg   string
	DescribedAt int64
	logger      *zap.Logger
}

type JobResult struct {
	JobID           uint
	Status          api.ComplianceReportJobStatus
	ReportCreatedAt int64
	Error           string
}

type SteampipeResultJson struct {
	Summary SteampipeResultSummaryJson `json:"summary"`
}
type SteampipeResultSummaryJson struct {
	Status SteampipeResultStatusJson `json:"status"`
}
type SteampipeResultStatusJson struct {
	Alarm int `json:"alarm"`
	Ok    int `json:"ok"`
	Info  int `json:"info"`
	Skip  int `json:"skip"`
	Error int `json:"error"`
}

func (j *Job) failed(msg string, args ...interface{}) JobResult {
	return JobResult{
		JobID:  j.JobID,
		Error:  fmt.Sprintf(msg, args...),
		Status: api.ComplianceReportJobCompletedWithFailure,
	}
}

func (j *Job) Do(vlt vault.SourceConfig, producer sarama.SyncProducer, topic string, config Config, logger *zap.Logger) JobResult {
	startTime := time.Now().Unix()

	cfg, err := vlt.Read(j.ConfigReg)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed("error: read source config: " + err.Error())
	}

	var accountID string
	switch j.SourceType {
	case source.CloudAWS:
		creds, err := AWSAccountConfigFromMap(cfg)
		if err != nil {
			DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
			return j.failed("error: AWSAccountConfigFromMap: " + err.Error())
		}
		accountID = creds.AccountID

		err = BuildSpecFile("aws", config.ElasticSearch, accountID)
		if err != nil {
			DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
			return j.failed("error: BuildSpecFile: " + err.Error())
		}
	case source.CloudAzure:
		creds, err := AzureSubscriptionConfigFromMap(cfg)
		if err != nil {
			DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
			return j.failed("error: AzureSubscriptionConfigFromMap: " + err.Error())
		}
		accountID = creds.SubscriptionID

		err = BuildSpecFile("azure", config.ElasticSearch, accountID)
		if err != nil {
			DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
			return j.failed("error: BuildSpecFile(azure): " + err.Error())
		}

		err = BuildSpecFile("azuread", config.ElasticSearch, accountID)
		if err != nil {
			DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
			return j.failed("error: BuildSpecFile(azuread) " + err.Error())
		}
	default:
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed("error: invalid source type")
	}

	resultFileName := fmt.Sprintf("result-%s-%d.json", accountID, time.Now().Unix())
	defer func() {
		err := os.Remove(resultFileName)
		if err != nil {
			logger.Error("Cannot remove file", zap.Error(err))
		}
	}()

	assignments, err := client.GetBenchmarkAssignmentsBySourceId(config.InventoryBaseUrl, j.SourceID)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed("error: Get benchmark assignments by source: " + err.Error())
	}

	var benchmakrs []string
	for _, assignment := range assignments {
		benchmakrs = append(benchmakrs, assignment.BenchmarkId)
	}
	if err := RunSteampipeCheckBenchmarks(j.SourceType, benchmakrs, resultFileName); err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed("error: RunSteampipeCheckBenchmarks: " + err.Error())
	}

	reports, err := ParseReport(resultFileName, j.JobID, j.ReportID, j.SourceID, j.DescribedAt, j.SourceType)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed("error: Parse report: " + err.Error())
	}

	var findings []es.Finding
	var summary Summary
	benchmarkID := ""
	serviceTotalChecks := map[string]int{}
	serviceNonCompliantChecks := map[string]int{}
	for _, report := range reports {
		if report.Type == ReportTypeBenchmark && report.Group.Level == 1 {
			benchmarkID = report.Group.ID
			summary = report.Group.Summary
		}

		if report.Type == ReportTypeResult {
			var name, location, resourceType string
			for _, dim := range report.Result.Result.Dimensions {
				if dim.Key == "metadata" {
					switch j.SourceType {
					case source.CloudAWS:
						var metadata aws.Metadata
						err = json.Unmarshal([]byte(dim.Value), &metadata)
						if err != nil {
							DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
							DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
							return j.failed("error: Parse metadata: " + err.Error())
						}
						name = metadata.Name
						location = metadata.Region
						resourceType = metadata.ResourceType
					case source.CloudAzure:
						var metadata azure.Metadata
						err = json.Unmarshal([]byte(dim.Value), &metadata)
						if err != nil {
							DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
							DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
							return j.failed("error: Parse metadata: " + err.Error())
						}
						name = metadata.Name
						location = metadata.Location
						resourceType = metadata.ResourceType
					}
				}
			}
			findings = append(findings, es.Finding{
				ID:                 uuid.New(),
				ReportJobID:        j.JobID,
				ReportID:           j.ReportID,
				ResourceID:         report.Result.Result.Resource,
				ResourceName:       name,
				ResourceLocation:   location,
				SourceID:           j.SourceID,
				ControlID:          report.Result.ControlId,
				ParentBenchmarkIDs: report.Result.ParentGroupIds,
				Status:             string(report.Result.Result.Status),
				DescribedAt:        report.DescribedAt,
			})

			sn := cloudservice.ServiceNameByResourceType(resourceType)
			serviceTotalChecks[sn]++
			if report.Result.Result.Status != ResultStatusOK {
				serviceNonCompliantChecks[sn]++
			}
		}
	}
	totalResource := summary.Status.OK + summary.Status.Info + summary.Status.Error + summary.Status.Alarm + summary.Status.Skip
	acr := AccountReport{
		SourceID:             j.SourceID,
		Provider:             j.SourceType,
		BenchmarkID:          benchmarkID,
		ReportJobId:          j.JobID,
		Summary:              summary,
		CreatedAt:            j.DescribedAt,
		TotalResources:       totalResource,
		TotalCompliant:       summary.Status.OK,
		CompliancePercentage: float64(summary.Status.OK) / float64(totalResource),
		AccountReportType:    es.AccountReportTypeInTime,
	}

	var sc []es.ServiceCompliancySummary
	for sn, total := range serviceTotalChecks {
		compliant := total
		if nc, ok := serviceNonCompliantChecks[sn]; ok {
			compliant = total - nc
		}
		sc = append(sc, es.ServiceCompliancySummary{
			ServiceName:          sn,
			TotalResources:       total,
			TotalCompliant:       compliant,
			CompliancePercentage: float64(compliant) / float64(total),
			CompliancySummary: es.CompliancySummary{
				CompliancySummaryType: es.CompliancySummaryTypeServiceSummary,
				ReportJobId:           j.JobID,
				Provider:              j.SourceType,
				DescribedAt:           j.DescribedAt,
			},
		})
	}

	acrLast := acr
	acrLast.AccountReportType = es.AccountReportTypeLast

	resourceStatus := map[string]ResultStatus{}
	for _, r := range reports {
		if r.Type == ReportTypeResult {
			current := resourceStatus[r.Result.Result.Resource]
			next := r.Result.Result.Status
			if current.SeverityLevel() < next.SeverityLevel() {
				resourceStatus[r.Result.Result.Resource] = next
			}
		}
	}
	nonCompliant := 0
	compliant := 0
	for _, status := range resourceStatus {
		if status == ResultStatusAlarm || status == ResultStatusError || status == ResultStatusInfo {
			nonCompliant++
		} else if status == ResultStatusOK {
			compliant++
		}
	}
	resource := kafka.ResourceCompliancyTrendResource{
		SourceID:                  j.SourceID.String(),
		SourceType:                j.SourceType,
		ComplianceJobID:           j.JobID,
		DescribedAt:               j.DescribedAt,
		CompliantResourceCount:    compliant,
		NonCompliantResourceCount: nonCompliant,
		ResourceSummaryType:       kafka.ResourceSummaryTypeCompliancyTrend,
	}

	var msgs []kafka.DescribedResource
	msgs = append(msgs, acr, acrLast, resource)
	for _, r := range reports {
		msgs = append(msgs, r)
	}
	for _, f := range findings {
		msgs = append(msgs, f)
	}
	for _, s := range sc {
		msgs = append(msgs, s)
	}
	err = kafka.DoSendToKafka(producer, topic, msgs, j.logger)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed("error: SendingToKafka: " + err.Error())
	}

	DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "successful").Observe(float64(time.Now().Unix() - startTime))
	DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "successful").Inc()
	return JobResult{
		JobID:           j.JobID,
		ReportCreatedAt: j.DescribedAt,
		Status:          api.ComplianceReportJobCompleted,
	}
}

func RunSteampipeCheckBenchmarks(sourceType source.Type, benchmarks []string, exportFileName string) error {
	workspaceDir := ""
	switch sourceType {
	case source.CloudAWS:
		workspaceDir = "/steampipe-mod-aws-compliance"
	case source.CloudAzure:
		workspaceDir = "/steampipe-mod-azure-compliance"
	default:
		return fmt.Errorf("invalid source type: %s", sourceType)
	}

	var args []string
	args = append(args, "check")
	if len(benchmarks) > 0 {
		args = append(args, benchmarks...)
	} else {
		args = append(args, "all")
	}
	args = append(args, "--export")
	args = append(args, exportFileName)
	args = append(args, "--workspace-chdir")
	args = append(args, workspaceDir)

	// steampipe will return total of alarms + errors as exit code
	// that would result in error on cmd.Run() output.
	// to prevent error on having alarms we ignore the error reported by cmd.Run()
	// exported json summery object is being used for discovering whether
	// there's an error or not
	_ = exec.Command("steampipe", args...).Run()

	data, err := ioutil.ReadFile(exportFileName)
	if err != nil {
		return err
	}
	if !json.Valid(data) {
		return fmt.Errorf("%s is invalid json file", exportFileName)
	}
	return nil
}

func BuildSpecFile(plugin string, config ElasticSearchConfig, accountID string) error {
	content := `
connection "` + plugin + `" {
  plugin = "` + plugin + `"
  addresses = ["` + config.Address + `"]
  username = "` + config.Username + `"
  password = "` + config.Password + `"
  accountID = "` + accountID + `"
}
`
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	filePath := dirname + "/.steampipe/config/" + plugin + ".spc"
	return ioutil.WriteFile(filePath, []byte(content), os.ModePerm)
}

type ComplianceReportCleanupJob struct {
	JobID uint // ComplianceReportJob ID
}

func (j ComplianceReportCleanupJob) Do(esClient *elasticsearch.Client) error {
	startTime := time.Now().Unix()

	ctx, cancel := context.WithTimeout(context.Background(), cleanupJobTimeout)
	defer cancel()

	fmt.Printf("Cleaning report with compliance_report_job_id of %d from index %s\n", j.JobID, ComplianceReportIndex)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"report_job_id": j.JobID,
			},
		},
	}

	indices := []string{
		ComplianceReportIndex,
	}

	resp, err := keibi.DeleteByQuery(ctx, esClient, indices, query,
		esClient.DeleteByQuery.WithRefresh(true),
		esClient.DeleteByQuery.WithConflicts("proceed"),
	)
	if err != nil {
		DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
		return err
	}

	if len(resp.Failures) != 0 {
		body, err := json.Marshal(resp)
		if err != nil {
			DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
			return err
		}

		DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
		return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	}

	fmt.Printf("Successfully delete %d reports\n", resp.Deleted)
	DoComplianceReportCleanupJobsDuration.WithLabelValues("successful").Observe(float64(time.Now().Unix() - startTime))
	DoComplianceReportCleanupJobsCount.WithLabelValues("successful").Inc()
	return nil
}
