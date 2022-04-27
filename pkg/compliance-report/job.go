package compliance_report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

type SourceType string

const (
	SourceCloudAWS    SourceType = "AWS"
	SourceCloudAzure  SourceType = "Azure"
	cleanupJobTimeout            = 5 * time.Minute
)

type ComplianceReportJobStatus string

const (
	ComplianceReportJobCreated              ComplianceReportJobStatus = "CREATED"
	ComplianceReportJobInProgress           ComplianceReportJobStatus = "IN_PROGRESS"
	ComplianceReportJobCompletedWithFailure ComplianceReportJobStatus = "COMPLETED_WITH_FAILURE"
	ComplianceReportJobCompleted            ComplianceReportJobStatus = "COMPLETED"
)

type Job struct {
	JobID      uint
	SourceID   uuid.UUID
	SourceType SourceType
	ConfigReg  string
	DescribeAt int64
	logger     *zap.Logger
}

type JobResult struct {
	JobID           uint
	Status          ComplianceReportJobStatus
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
		Status: ComplianceReportJobCompletedWithFailure,
	}
}

func (j *Job) Do(vlt vault.SourceConfig, producer sarama.SyncProducer, topic string, config Config, logger *zap.Logger) JobResult {
	cfg, err := vlt.Read(j.ConfigReg)
	if err != nil {
		return j.failed("error: read source config: " + err.Error())
	}

	var accountID string
	switch j.SourceType {
	case SourceCloudAWS:
		creds, err := AWSAccountConfigFromMap(cfg)
		if err != nil {
			return j.failed("error: AWSAccountConfigFromMap: " + err.Error())
		}
		accountID = creds.AccountID

		err = BuildSpecFile("aws", config.ElasticSearch, accountID)
		if err != nil {
			return j.failed("error: BuildSpecFile: " + err.Error())
		}
	case SourceCloudAzure:
		creds, err := AzureSubscriptionConfigFromMap(cfg)
		if err != nil {
			return j.failed("error: AzureSubscriptionConfigFromMap: " + err.Error())
		}
		accountID = creds.SubscriptionID

		err = BuildSpecFile("azure", config.ElasticSearch, accountID)
		if err != nil {
			return j.failed("error: BuildSpecFile(azure): " + err.Error())
		}

		err = BuildSpecFile("azuread", config.ElasticSearch, accountID)
		if err != nil {
			return j.failed("error: BuildSpecFile(azuread) " + err.Error())
		}
	default:
		return j.failed("error: invalid source type")
	}

	resultFileName := fmt.Sprintf("result-%s-%d.json", accountID, time.Now().Unix())
	defer func() {
		err := os.Remove(resultFileName)
		if err != nil {
			logger.Error("Cannot remove file", zap.Error(err))
		}
	}()

	err = RunSteampipeCheckAll(j.SourceType, resultFileName)
	if err != nil {
		return j.failed("error: RunSteampipeCheckAll: " + err.Error())
	}

	reports, err := ParseReport(resultFileName, j.JobID, j.SourceID, j.DescribeAt, j.SourceType)
	if err != nil {
		return j.failed("error: Parse report: " + err.Error())
	}

	var summary Summary
	benchmarkID := ""
	for _, report := range reports {
		if report.Type == ReportTypeBenchmark && report.Group.Level == 1 {
			benchmarkID = report.Group.ID
			summary = report.Group.Summary
		}
	}
	totalResource := summary.Status.OK + summary.Status.Info + summary.Status.Error + summary.Status.Alarm + summary.Status.Skip
	acr := AccountReport{
		SourceID:             j.SourceID,
		Provider:             j.SourceType,
		BenchmarkID:          benchmarkID,
		ReportJobId:          j.JobID,
		Summary:              summary,
		CreatedAt:            j.DescribeAt,
		TotalResources:       totalResource,
		TotalCompliant:       summary.Status.OK,
		CompliancePercentage: float64(summary.Status.OK) / float64(totalResource),
	}

	err = doSendToKafka(producer, topic, reports, acr, j.logger)
	if err != nil {
		return j.failed("error: SendingToKafka: " + err.Error())
	}

	return JobResult{
		JobID:           j.JobID,
		ReportCreatedAt: j.DescribeAt,
		Status:          ComplianceReportJobCompleted,
	}
}

func doSendToKafka(producer sarama.SyncProducer, topic string, reports []Report, accountReport AccountReport, logger *zap.Logger) error {
	var msgs []*sarama.ProducerMessage
	for _, v := range reports {
		msg, err := v.AsProducerMessage()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to convert report[%v] to Kafka ProducerMessage, ignoring...", v))
			continue
		}

		// Override the topic
		msg.Topic = topic

		msgs = append(msgs, msg)
	}

	msg, err := accountReport.AsProducerMessage()
	if err != nil {
		fmt.Printf("Failed to convert accountReport[%v] to Kafka ProducerMessage, ignoring...", accountReport.SourceID)
	} else {
		// Override the topic
		msg.Topic = topic
		msgs = append(msgs, msg)
	}

	if len(msgs) == 0 {
		return nil
	}

	if err := producer.SendMessages(msgs); err != nil {
		if errs, ok := err.(sarama.ProducerErrors); ok {
			for _, e := range errs {
				fmt.Printf("Failed to persist resource[%s] in kafka topic[%s]: %s\nMessage: %v\n", e.Msg.Key, e.Msg.Topic, e.Error(), e.Msg)
			}
		}

		return err
	}

	return nil
}

func RunSteampipeCheckAll(sourceType SourceType, exportFileName string) error {
	workspaceDir := ""

	switch sourceType {
	case SourceCloudAWS:
		workspaceDir = "/steampipe-mod-aws-compliance"
	case SourceCloudAzure:
		workspaceDir = "/steampipe-mod-azure-compliance"
	default:
		return errors.New("invalid source type")
	}

	cmd := exec.Command("steampipe", "check", "all",
		"--export", exportFileName, "--workspace-chdir", workspaceDir)
	// steampipe will return total of alarms + errors as exit code
	// that would result in error on cmd.Run() output.
	// to prevent error on having alarms we ignore the error reported by cmd.Run()
	// exported json summery object is being used for discovering whether
	// there's an error or not
	_ = cmd.Run()

	contentBytes, err := ioutil.ReadFile(exportFileName)
	if err != nil {
		return err
	}

	var v SteampipeResultJson
	err = json.Unmarshal(contentBytes, &v)
	if err != nil {
		return err
	}

	//if v.Summary.Status.Error > 0 {
	//	return fmt.Errorf("steampipe returned %d errors", v.Summary.Status.Error)
	//}
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
		return err
	}

	if len(resp.Failures) != 0 {
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	}

	fmt.Printf("Successfully delete %d reports\n", resp.Deleted)
	return nil
}
