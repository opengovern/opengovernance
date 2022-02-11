package steampipe

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe/mock"
	"io/ioutil"
	"os"
	"testing"
)

func TestJob_Do(t *testing.T) {
	job := Job {
		SourceID: "",
		SourceType: describe.SourceCloudAzure,
	}

	s3 := mock.NewMockedS3Client()

	result := job.Do(s3, Config{
		S3Client: S3ClientConfig{},
		RabbitMQ: RabbitMQConfig{},
		ElasticSearch: ElasticSearchConfig{},
	})

	assert.Equal(t, "successful", result.Result)
	assert.NotEmpty(t, result.S3ResultURL)
}

func TestBuildSpecFile(t *testing.T) {
	err := BuildSpecFile("test", ElasticSearchConfig{
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
