package test

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	compliancereport "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/dockertest"
	"gopkg.in/Shopify/sarama.v1"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestJob_Do(t *testing.T) {
	job := compliancereport.Job{
		JobID:      100,
		SourceID:   uuid.New(),
		SourceType: compliancereport.SourceCloudAzure,
		ConfigReg:  "azure",
	}

	vault := SourceConfigMock{}
	kafka := dockertest.StartupKafka(t)
	es := dockertest.StartupElasticSearch(t)

	topic := "test"
	result := job.Do(vault, kafka.Producer, topic, compliancereport.Config{
		RabbitMQ: compliancereport.RabbitMQConfig{},
		ElasticSearch: compliancereport.ElasticSearchConfig{
			Address: es.Address,
		},
	})
	assert.Equal(t, "", result.Error)

	conf := sarama.NewConfig()
	conf.Producer.Retry.Max = 1
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Return.Successes = true
	conf.Metadata.Full = true
	conf.ClientID = "kafka_client"

	doneChannel := make(chan bool)
	consumer, err := sarama.NewConsumer(strings.Split(kafka.Address, ","), conf)
	assert.Nil(t, err)

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	assert.Nil(t, err)

	count := 0
ConsumerLoop:
	for {
		select {
		case _ = <-partitionConsumer.Messages():
			count++
			if count == 1356 {
				break ConsumerLoop
			}
		case <-doneChannel:
			break ConsumerLoop
		}
	}

	go func() {
		time.Sleep(30 * time.Second)
		doneChannel <- true
	}()

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
  accountID = "sourceID"
}
`, string(content))

	_ = os.Remove(userHome + "/.steampipe/config/test.spc")
}
