package compliance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"gitlab.com/keibiengine/keibi-engine/pkg/config"

	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"go.uber.org/zap"
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
	JobID         uint
	ScheduleJobID uint
	SourceID      uuid.UUID
	BenchmarkID   string
	SourceType    source.Type
	ConfigReg     string
	DescribedAt   int64
	logger        *zap.Logger
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

func (j *Job) Do(w *Worker) JobResult {
	startTime := time.Now().Unix()

	cfg, err := w.vault.Read(j.ConfigReg)
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

		err = BuildSpecFile("aws", w.config.ElasticSearch, accountID)
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

		err = BuildSpecFile("azure", w.config.ElasticSearch, accountID)
		if err != nil {
			DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
			DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
			return j.failed("error: BuildSpecFile(azure): " + err.Error())
		}

		err = BuildSpecFile("azuread", w.config.ElasticSearch, accountID)
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

	modPath, err := j.BuildMod(w.db)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed(err.Error())
	}

	resultFileName, err := j.RunSteampipeCheckFor(modPath)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed(err.Error())
	}

	evaluatedAt := time.Now().UnixMilli()
	msgs, err := j.ParseResult(w.onboardClient, resultFileName, evaluatedAt)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed(err.Error())
	}

	err = kafka.DoSend(w.kfkProducer, w.kfkTopic, msgs, j.logger)
	if err != nil {
		DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "failure").Observe(float64(time.Now().Unix() - startTime))
		DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "failure").Inc()
		return j.failed("error: SendingToKafka: " + err.Error())
	}

	DoComplianceReportJobsDuration.WithLabelValues(string(j.SourceType), "successful").Observe(float64(time.Now().Unix() - startTime))
	DoComplianceReportJobsCount.WithLabelValues(string(j.SourceType), "successful").Inc()
	return JobResult{
		JobID:           j.JobID,
		ReportCreatedAt: evaluatedAt,
		Status:          api.ComplianceReportJobCompleted,
	}
}

func (j *Job) RunSteampipeCheckFor(modPath string) (string, error) {
	exportFileName := "result.json"

	var args []string
	args = append(args, "check")
	args = append(args, "all")
	args = append(args, "--export")
	args = append(args, exportFileName)
	args = append(args, "--workspace-chdir")
	args = append(args, modPath)

	// steampipe will return total of alarms + errors as exit code
	// that would result in error on cmd.Run() output.
	// to prevent error on having alarms we ignore the error reported by cmd.Run()
	// exported json summery object is being used for discovering whether
	// there's an error or not
	_ = exec.Command("steampipe", args...).Run()

	data, err := ioutil.ReadFile(exportFileName)
	if err != nil {
		return exportFileName, err
	}
	if !json.Valid(data) {
		return exportFileName, fmt.Errorf("%s is invalid json file", exportFileName)
	}
	return exportFileName, nil
}

func (j *Job) BuildMod(db Database) (string, error) {
	mod := steampipe.Mod{
		ID:    fmt.Sprintf("mod-%d", j.JobID),
		Title: fmt.Sprintf("Compliance worker mod for job %d", j.JobID),
	}

	b, err := db.GetBenchmark(j.BenchmarkID)
	if err != nil {
		return "", err
	}
	if b == nil {
		return "", errors.New("benchmark not found")
	}

	var controls []steampipe.Control
	for _, p := range b.Policies {
		tags := map[string]string{}
		for _, tag := range p.Tags {
			tags[tag.Key] = tag.Value
		}
		controls = append(controls, steampipe.Control{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			Severity:    p.Severity,
			Tags:        tags,
			SQL:         p.QueryToRun,
		})
	}
	benchmark := steampipe.Benchmark{
		ID:          b.ID,
		Title:       b.Title,
		Description: b.Description,
		Children:    controls,
	}

	content := mod.String() + "\n\n" + benchmark.String()

	_ = os.Mkdir(mod.ID, os.ModePerm)
	filename := mod.ID + "/mod.sp"
	err = ioutil.WriteFile(filename, []byte(content), os.ModePerm)

	return filename, err
}

func (j *Job) ParseResult(onboardClient client.OnboardServiceClient, resultFilename string, evaluatedAt int64) ([]kafka.Doc, error) {
	content, err := ioutil.ReadFile(resultFilename)
	if err != nil {
		return nil, err
	}

	var root api.Group
	err = json.Unmarshal(content, &root)
	if err != nil {
		return nil, err
	}

	findings := j.ExtractFindings(root, evaluatedAt)

	var sourceIDs []string
	for _, f := range findings {
		sourceIDs = append(sourceIDs, f.SourceID.String())
	}

	sources, err := onboardClient.GetSources(&httpclient.Context{UserRole: api2.ViewerRole}, sourceIDs)
	if err != nil {
		return nil, err
	}

	var res []kafka.Doc
	for _, f := range findings {
		for _, s := range sources {
			if s.ID == f.SourceID {
				f.ConnectionProviderID = s.ConnectionID
				f.ConnectionProviderName = s.ConnectionName
				break
			}
		}
		res = append(res, f)
	}
	return res, nil
}

func (j *Job) ExtractFindings(root api.Group, evaluatedAt int64) []es.Finding {
	var findings []es.Finding
	for _, c := range root.Controls {
		for _, r := range c.Results {
			var resourceName, resourceLocation, resourceType string
			for _, d := range r.Dimensions {
				if d.Key == "name" {
					resourceName = d.Value
				} else if d.Key == "location" {
					resourceLocation = d.Value
				} else if d.Key == "resourceType" {
					resourceType = d.Value
				}
			}

			findings = append(findings, es.Finding{
				ComplianceJobID:        j.JobID,
				ScheduleJobID:          j.ScheduleJobID,
				ResourceID:             r.Resource,
				ResourceName:           resourceName,
				ResourceType:           resourceType,
				ServiceName:            cloudservice.ServiceNameByResourceType(resourceType),
				Category:               cloudservice.CategoryByResourceType(resourceType),
				ResourceLocation:       resourceLocation,
				Reason:                 r.Reason,
				Status:                 r.Status,
				DescribedAt:            j.DescribedAt,
				EvaluatedAt:            evaluatedAt,
				SourceID:               j.SourceID,
				ConnectionProviderID:   "",
				ConnectionProviderName: "",
				SourceType:             j.SourceType,
				BenchmarkID:            j.BenchmarkID,
				PolicyID:               c.ControlId,
				PolicySeverity:         c.Severity,
			})
		}
	}

	for _, g := range root.Groups {
		findings = append(findings, j.ExtractFindings(g, evaluatedAt)...)
	}
	return findings
}

func BuildSpecFile(plugin string, config config.ElasticSearch, accountID string) error {
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
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	return ioutil.WriteFile(filePath, []byte(content), os.ModePerm)
}

type ComplianceReportCleanupJob struct {
	JobID uint // ComplianceReportJob ID
}

func (j ComplianceReportCleanupJob) Do(esClient *elasticsearch.Client) error {
	return nil
	//startTime := time.Now().Unix()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), cleanupJobTimeout)
	//defer cancel()
	//
	//fmt.Printf("Cleaning report with compliance_report_job_id of %d from index %s\n", j.JobID, es.ComplianceReportIndex)
	//
	//query := map[string]interface{}{
	//	"query": map[string]interface{}{
	//		"match": map[string]interface{}{
	//			"report_job_id": j.JobID,
	//		},
	//	},
	//}
	//
	//indices := []string{
	//	es.ComplianceReportIndex,
	//}
	//
	//resp, err := keibi.DeleteByQuery(ctx, esClient, indices, query,
	//	esClient.DeleteByQuery.WithRefresh(true),
	//	esClient.DeleteByQuery.WithConflicts("proceed"),
	//)
	//if err != nil {
	//	DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
	//	DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
	//	return err
	//}
	//
	//if len(resp.Failures) != 0 {
	//	body, err := json.Marshal(resp)
	//	if err != nil {
	//		DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
	//		DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
	//		return err
	//	}
	//
	//	DoComplianceReportCleanupJobsDuration.WithLabelValues("failure").Observe(float64(time.Now().Unix() - startTime))
	//	DoComplianceReportCleanupJobsCount.WithLabelValues("failure").Inc()
	//	return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	//}
	//
	//fmt.Printf("Successfully delete %d reports\n", resp.Deleted)
	//DoComplianceReportCleanupJobsDuration.WithLabelValues("successful").Observe(float64(time.Now().Unix() - startTime))
	//DoComplianceReportCleanupJobsCount.WithLabelValues("successful").Inc()
	//return nil
}
