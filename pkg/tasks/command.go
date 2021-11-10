package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/streadway/amqp"
	"gitlab.com/anil94/golang-aws-inventory/pkg/aws"
	"gitlab.com/anil94/golang-aws-inventory/pkg/azure"
	"gopkg.in/Shopify/sarama.v1"
)

var (
	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	KafkaService = os.Getenv("KAFKA_SERVICE")
)

func PublishCommand() *cobra.Command {
	var (
		publisherId string
		file        string
	)

	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case publisherId == "":
				return errors.New("missing required flag 'id'")
			case file == "":
				return errors.New("missing required flag 'file'")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			content, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			var cfg Config
			if err := json.Unmarshal(content, &cfg); err != nil {
				return fmt.Errorf("config: %s", err)
			}

			amqpUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/", RabbitMQUsername, RabbitMQPassword, RabbitMQService, RabbitMQPort)

			queue, err := NewDescribeQueue(amqpUrl)
			if err != nil {
				return err
			}
			defer queue.Close()

			for _, entry := range cfg.Entries {
				var types []string

				var account string
				switch entry.Type {
				case CloudTypeAWS:
					account = entry.AWS.AccountId
					types = aws.ListResourceTypes()
				case CloudTypeAzure:
					account = strings.Join(entry.Azure.Subscriptions, ",")
					types = azure.ListResourceTypes()
				default:
					fmt.Printf("invalid entry type '%s'; won't publish task!", entry.Type)
					continue
				}

				for _, rType := range types {
					fmt.Printf("Publishing task to describe resource type '%s' from account '%s' of cloud '%s'\n", rType, account, entry.Type)
					body, err := json.Marshal(Message{
						ResourceType: rType,
						AWS:          entry.AWS,
						Azure:        entry.Azure,
					})
					if err != nil {
						fmt.Println("Failed!")
						continue
					}

					err = queue.Publish(
						amqp.Publishing{
							ContentType: "applicatin/json",
							Body:        body,
							AppId:       publisherId,
							Timestamp:   time.Now(),
						})
					if err != nil {
						fmt.Println("Failed!")
						continue
					}

					fmt.Println("Successfully published task!")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "The file containing cloud account entries and the credentials")
	cmd.Flags().StringVar(&publisherId, "id", "", "The publisher id is used to annotate the published messages")

	return cmd
}

func ConsumeCommand() *cobra.Command {
	var (
		consumerId string
		topic      string
	)
	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case consumerId == "":
				return errors.New("missing required flag 'id'")
			case topic == "":
				return errors.New("missing required flag 'topic'")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			amqpUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/", RabbitMQUsername, RabbitMQPassword, RabbitMQService, RabbitMQPort)

			queue, err := NewDescribeQueue(amqpUrl)
			if err != nil {
				return err
			}
			defer queue.Close()

			tasks, err := queue.Consume(consumerId)
			if err != nil {
				return err
			}

			cfg := sarama.NewConfig()
			cfg.Producer.RequiredAcks = sarama.WaitForAll
			cfg.Producer.Return.Successes = true
			cfg.Version = sarama.V2_1_0_0

			producer, err := sarama.NewSyncProducer(strings.Split(KafkaService, ","), cfg)
			if err != nil {
				return err
			}
			defer producer.Close()

			go func() {
				for task := range tasks {
					inventory, err := doDescribe(task)
					if err != nil {
						task.Nack(false, false) // TODO: Maybe Requeue for certain errors
					}

					sendToKafka(producer, topic, inventory)
					task.Ack(false)
				}
			}()

			fmt.Printf("Waiting indefinitly for messages. To exit press CTRL+C")

			forever := make(chan bool)
			<-forever
			return nil
		},
	}

	cmd.Flags().StringVar(&consumerId, "id", "", "The consumer id")
	cmd.Flags().StringVarP(&topic, "topic", "t", "", "The kafka topic where the resources are published.")

	return cmd
}

func doDescribe(task amqp.Delivery) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var message Message
	err := json.Unmarshal(task.Body, &message)
	if err != nil {
		return nil, fmt.Errorf("unmarshall task: %w", err)
	}

	fmt.Printf("Proccessing Message %s, RosourceType %s\n", task.MessageId, message.ResourceType)

	var inventory []map[string]interface{}
	switch {
	case message.AWS != nil:
		output, err := aws.GetResources(
			ctx,
			message.ResourceType,
			message.AWS.AccountId,
			message.AWS.Regions,
			message.AWS.Credentials.AccessKey,
			message.AWS.Credentials.SecretKey,
			message.AWS.Credentials.SessionToken,
			message.AWS.Credentials.AssumeRoleARN,
			false,
		)
		if err != nil {
			return nil, fmt.Errorf("describe AWS resources: %w", err)
		}

		for region, resources := range output.Resources {
			for _, resource := range resources {
				if resource == nil {
					continue
				}

				inventory = append(inventory, map[string]interface{}{
					output.Metadata.ResourceType: resource,
					"CloudType":                  CloudTypeAWS,
					"ResourceType":               output.Metadata.ResourceType,
					"AccountId":                  output.Metadata.AccountId,
					"Region":                     region,
				})
			}
		}
	case message.Azure != nil:
		output, err := azure.GetResources(
			ctx,
			message.ResourceType,
			message.Azure.Subscriptions,
			message.Azure.Credentials.TenantID,
			message.Azure.Credentials.ClientID,
			message.Azure.Credentials.ClientSecret,
			message.Azure.Credentials.CertificatePath,
			message.Azure.Credentials.CertificatePass,
			message.Azure.Credentials.Username,
			message.Azure.Credentials.Password,
			string(azure.AuthEnv),
			"",
		)
		if err != nil {
			return nil, fmt.Errorf("describe Azure resources: %w", err)
		}

		for _, resource := range output.Resources {
			if resource == nil {
				continue
			}

			inventory = append(inventory, map[string]interface{}{
				output.Metadata.ResourceType: resource,
				"CloudType":                  CloudTypeAWS,
				"ResourceType":               output.Metadata.ResourceType,
				"SubscriptionIds":            strings.Join(output.Metadata.SubscriptionIds, ","),
			})
		}
	}

	return inventory, nil
}

func sendToKafka(producer sarama.SyncProducer, topic string, inventory []map[string]interface{}) {
	var msgs []*sarama.ProducerMessage
	for _, v := range inventory {
		value, err := json.Marshal(v)
		if err != nil {
			fmt.Printf("failed to marshal %v", v)
			continue
		}

		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(value),
		})
	}

	if len(msgs) != 0 {
		if err := producer.SendMessages(msgs); err != nil {
			if errs, ok := err.(sarama.ProducerErrors); ok {
				for _, e := range errs {
					fmt.Printf("failed to persist resource in kafka: %s\nMessage: %v\n", e.Error(), e.Msg)
				}
			}
		}
	}
}
