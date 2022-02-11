package steampipe

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

type Job struct {
	SourceID   string // SubscriptionID for Azure or AccountID for AWS
	SourceType describe.SourceType
}

type JobResult struct {
	S3ResultURL string
	Result      string
}

func (j *Job) Do(s3Client s3iface.S3API, config Config) JobResult {
	if j.SourceType == describe.SourceCloudAWS {
		err := BuildSpecFile("aws", config.ElasticSearch, j.SourceID)
		if err != nil {
			return JobResult{Result: "error: failed to build aws spec file due to " + err.Error()}
		}
	} else if j.SourceType == describe.SourceCloudAzure {
		err := BuildSpecFile("azure", config.ElasticSearch, j.SourceID)
		if err != nil {
			return JobResult{Result: "error: failed to build azure spec file due to " + err.Error()}
		}

		err = BuildSpecFile("azuread", config.ElasticSearch, j.SourceID)
		if err != nil {
			return JobResult{Result: "error: failed to build azuread spec file due to " + err.Error()}
		}
	} else {
		return JobResult{Result: "error: invalid source type"}
	}

	resultFileName := fmt.Sprintf("result-%s-%d.html", j.SourceID, time.Now().Unix())

	err := RunSteampipeCheckAll(j.SourceType, resultFileName)
	if err != nil {
		return JobResult{Result: "error: failed to run check all due to " + err.Error()}
	}

	file, err := os.OpenFile(resultFileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return JobResult{Result: "error: failed to open result file due to " + err.Error()}
	}

	urlStr, err := UploadIntoS3Storage(s3Client, config.S3Client.Bucket, resultFileName, file)
	if err != nil {
		return JobResult{Result: "error: failed to upload result file into s3 storage due to " + err.Error()}
	}

	return JobResult{S3ResultURL: urlStr, Result: "successful"}
}

func RunSteampipeCheckAll(sourceType describe.SourceType, exportFileName string) error {
	workspaceDir := ""

	switch sourceType {
	case describe.SourceCloudAWS:
		workspaceDir = "/steampipe-mod-aws-compliance"
	case describe.SourceCloudAzure:
		workspaceDir = "/steampipe-mod-azure-compliance"
	default:
		return errors.New("invalid source type")
	}

	cmd := exec.Command( "steampipe", "check", "all",
		"--export", exportFileName, "--workspace-chdir", workspaceDir)

	return cmd.Run()
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

func BuildSpecFile(plugin string, config ElasticSearchConfig, sourceID string) error {
	content := `
connection "` + plugin + `" {
  plugin = "` + plugin + `"
  addresses = ["` + config.Address + `"]
  username = "` + config.Username + `"
  password = "` + config.Password + `"
  sourceID = "` + sourceID + `"
}
`
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	filePath := dirname + "/.steampipe/config/" + plugin + ".spc"
	return ioutil.WriteFile(filePath, []byte(content), os.ModePerm)
}
