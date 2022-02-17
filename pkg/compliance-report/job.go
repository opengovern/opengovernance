package compliance_report

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
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
	SourceType SourceType
	ConfigReg  string
}

type JobResult struct {
	JobID       uint
	S3ResultURL string
	Status      ComplianceReportJobStatus
	Error       string
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
		Error:  fmt.Sprintf(msg, args),
		Status: ComplianceReportJobCompletedWithFailure,
	}
}

func (j *Job) Do(vlt vault.Keibi, s3Client s3iface.S3API, config Config) JobResult {
	cfg, err := vlt.ReadSourceConfig(j.ConfigReg)
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

	err = RunSteampipeCheckAll(j.SourceType, resultFileName)
	if err != nil {
		return j.failed("error: RunSteampipeCheckAll: " + err.Error())
	}

	file, err := os.OpenFile(resultFileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return j.failed("error: OpenFile: " + err.Error())
	}

	urlStr, err := UploadIntoS3Storage(s3Client, config.S3Client.Bucket, resultFileName, file)
	if err != nil {
		return j.failed("error: UploadIntoS3Storage: " + err.Error())
	}

	return JobResult{
		JobID:       j.JobID,
		S3ResultURL: urlStr,
		Status:      ComplianceReportJobCompleted,
	}
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

	if v.Summary.Status.Error > 0 {
		return fmt.Errorf("steampipe returned %d errors", v.Summary.Status.Error)
	}
	return nil
}

func UploadIntoS3Storage(s3Client s3iface.S3API, bucketName string, keyName string, contentReader io.ReadSeeker) (string, error) {
	object := s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
		Body:   contentReader,
		ACL:    aws.String("private"),
	}
	_, err := s3Client.PutObject(&object)
	if err != nil {
		return "", err
	}

	req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	})
	urlStr, err := req.Presign(5 * time.Minute)
	if err != nil {
		return "", err
	}

	return urlStr, nil
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
