package test

import (
	"github.com/stretchr/testify/assert"
	compliancereport "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"io/ioutil"
	"os"
	"testing"
)

func TestJob_Do(t *testing.T) {
	job := compliancereport.Job{
		JobID:      0,
		SourceType: compliancereport.SourceCloudAzure,
		ConfigReg:  "azure",
	}

	vault := SourceConfigMock{}
	s3 := NewMockedS3Client()

	result := job.Do(vault, s3, compliancereport.Config{
		S3Client:      compliancereport.S3ClientConfig{},
		RabbitMQ:      compliancereport.RabbitMQConfig{},
		ElasticSearch: compliancereport.ElasticSearchConfig{},
	})

	assert.Equal(t, "", result.Error)
	assert.NotEmpty(t, result.S3ResultURL)
}

func TestBuildSpecFile(t *testing.T) {
	err := compliancereport.BuildSpecFile("test", compliancereport.ElasticSearchConfig{
		Address:  "test-address:1000",
		Username: "username",
		Password: "password",
	}, "sourceID")
	assert.Nil(t, err)

	userHome, err := os.UserHomeDir()
	assert.Nil(t, err)

	content, err := ioutil.ReadFile(userHome + "/.steampipe/config/test.spc")
	assert.Nil(t, err)

	assert.Equal(t, `
connection "test" {
  plugin = "test"
  addresses = ["test-address:1000"]
  username = "username"
  password = "password"
  sourceID = "sourceID"
}
`, string(content))

	_ = os.Remove(userHome + "/.steampipe/config/test.spc")
}
