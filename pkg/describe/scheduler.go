package describe

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/anil94/golang-aws-inventory/pkg/aws"
	"gitlab.com/anil94/golang-aws-inventory/pkg/azure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	JobCompletionInterval   = 1 * time.Minute
	JobSchedulingInterval   = 1 * time.Minute
	JobTimeoutCheckInterval = 15 * time.Minute
)

type Scheduler struct {
	id string
	db Database
	// jobQueue is used to publish jobs to be performed by the workers.
	jobQueue *Queue
	// jobResultQueue is used to consume the job results returned by the workers.
	jobResultQueue *Queue
	// sourceQueue is used to consume source updates by the onboarding service.
	sourceQueue *Queue
}

func InitializeScheduler(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	describeJobQueue string,
	describeJobResultQueue string,
	sourceQueue string,
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
) (s *Scheduler, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	s = &Scheduler{id: id}
	defer func() {
		if err != nil && s != nil {
			s.Stop()
		}
	}()

	fmt.Println("Initializing the scheduler")

	qCfg := QueueConfig{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobQueue
	qCfg.Queue.Durable = true
	describeQueue, err := NewQueue(qCfg)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the describe jobs queue: ", describeJobQueue)
	s.jobQueue = describeQueue

	qCfg = QueueConfig{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobResultQueue
	qCfg.Queue.Durable = true
	describeResultsQueue, err := NewQueue(qCfg)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the describe job results queue: ", describeJobResultQueue)
	s.jobResultQueue = describeResultsQueue

	qCfg = QueueConfig{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = sourceQueue
	qCfg.Queue.Durable = true
	sourceEventsQueue, err := NewQueue(qCfg)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the source events queue: ", sourceQueue)
	s.sourceQueue = sourceEventsQueue

	dsn := fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`,
		postgresHost,
		postgresPort,
		postgresUsername,
		postgresPassword,
		postgresDb,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the postgres database: ", postgresDb)
	s.db = Database{orm: db}

	return s, nil
}

func (s *Scheduler) Run() error {
	err := s.db.orm.AutoMigrate(&Source{}, &DescribeSourceJob{}, &DescribeResourceJob{})
	if err != nil {
		return err
	}

	go s.RunSourceEventsConsumer()
	go s.RunJobCompletionUpdater()
	go s.RunDescribeScheduler()

	// RunJobResultsConsumer shouldn't return.
	// If it does, indicates a failure with consume
	return s.RunJobResultsConsumer()
}

func (s *Scheduler) RunJobCompletionUpdater() {
	t := time.NewTicker(JobCompletionInterval)
	defer t.Stop()

	for ; ; <-t.C {
		results, err := s.db.QueryInProgressDescribedSourceJobGroupByDescribeResourceJobStatus()
		if err != nil {
			fmt.Println("Error finding the DescribeSourceJobs: ", err.Error())
			continue
		}

		jobIDToStatus := make(map[uint]map[DescribeResourceJobStatus]int)
		for _, v := range results {
			if _, ok := jobIDToStatus[v.DescribeSourceJobID]; !ok {
				jobIDToStatus[v.DescribeSourceJobID] = map[DescribeResourceJobStatus]int{
					DescribeResourceJobCreated:   0,
					DescribeResourceJobQueued:    0,
					DescribeResourceJobFailed:    0,
					DescribeResourceJobSucceeded: 0,
				}
			}

			jobIDToStatus[v.DescribeSourceJobID][v.DescribeResourceJobStatus] = v.DescribeResourceJobCount
		}

		for id, status := range jobIDToStatus {
			// If any CREATED or QUEUED, job is still in progress
			if status[DescribeResourceJobCreated] > 0 ||
				status[DescribeResourceJobQueued] > 0 {
				continue
			}

			// If any FAILURE, job is completed with failure
			if status[DescribeResourceJobFailed] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, DescribeSourceJobCompletedWithFailure)
				if err != nil {
					fmt.Printf("Error updating DescribeSourceJob %d status to %s: %s\n", id, DescribeSourceJobCompletedWithFailure, err.Error())
				}

				continue
			}

			// If the rest is SUCCEEDED, job has completed with no failure
			if status[DescribeResourceJobSucceeded] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, DescribeSourceJobCompleted)
				if err != nil {
					fmt.Printf("Error updating DescribeSourceJob %d status to %s: %s\n", id, DescribeSourceJobCompleted, err.Error())
				}

				continue
			}
		}
	}
}

func (s *Scheduler) RunDescribeScheduler() {
	fmt.Println("Scheduling describe jobs on a timer")
	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		sources, err := s.db.QuerySourcesDueForDescribe()
		if err != nil {
			fmt.Printf("Error finding the next sources to create DescribeSourceJob: %s\n", err.Error())
			continue
		}

		for _, source := range sources {
			fmt.Printf("Source[%s] is due for a describe. Creating a job now\n", source.ID)

			daj := newDescribeSourceJob(source)
			err := s.db.CreateDescribeSourceJob(&daj)
			if err != nil {
				fmt.Printf("Failed to create DescribeSourceJob[%d] for Source[%d]: %s\n", daj.ID, source.ID, err.Error())
				continue
			}

			enqueueDescribeResourceJobs(s.db, s.jobQueue, source, daj)

			err = s.db.UpdateDescribeSourceJob(daj.ID, DescribeSourceJobInProgress)
			if err != nil {
				fmt.Printf("Failed to update DescribeSourceJob[%d]: %s\n", daj.ID, err.Error())
			}
			daj.Status = DescribeSourceJobInProgress

			err = s.db.UpdateSourceDescribed(source.ID)
			if err != nil {
				fmt.Printf("Failed to update Source[%d]: %s\n", source.ID, err.Error())
			}
			daj.Status = DescribeSourceJobInProgress
		}
	}
}

// Consume events from the source queue. Based on the action of the event,
// update the list of sources that need to be described. Either create a source
// or update/delete the source.
func (s *Scheduler) RunSourceEventsConsumer() error {
	fmt.Println("Consuming messages from SourceEvents queue")
	msgs, err := s.sourceQueue.Consume(s.id)
	if err != nil {
		return err
	}

	for msg := range msgs {
		var event SourceEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			fmt.Printf("Failed to unmarshal SourceEvent: %s\n", err.Error())
			msg.Nack(false, false)
			continue
		}

		err := ProcessSourceAction(s.db, event)
		if err != nil {
			fmt.Printf("Failed to process event for Source[%s]: %s", event.SourceID, err)
			msg.Nack(false, false)
			continue
		}

		msg.Ack(false)
	}

	return fmt.Errorf("source events queue channel is closed")
}

// RunJobResultsConsumer consumes messages from the jobResult queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunJobResultsConsumer() error {
	fmt.Println("Consuming messages from the JobResults queue")

	msgs, err := s.jobResultQueue.Consume(s.id)
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result JobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				fmt.Printf("Failed to unmarshal DescribeResourceJob results: %s\n", err.Error())
				msg.Nack(false, false)
				continue
			}

			fmt.Printf("Processing JobResult for Job[%d]: job status is %s\n", result.JobID, result.Status)
			err := s.db.UpdateDescribeResourceJobStatus(result.JobID, result.Status, result.Error)
			if err != nil {
				fmt.Printf("Failed to update the status of DescribeResourceJob[%d]: %s\n", result.JobID, err.Error())
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
		case <-t.C:
			err := s.db.UpdateDescribeResourceJobsTimedOut()
			if err != nil {
				fmt.Printf("Failed to update timed out DescribeResourceJobs: %s\n", err.Error())
			}
		}
	}
}

func (s *Scheduler) Stop() {
	if s.jobQueue != nil {
		s.jobQueue.Close()
		s.jobQueue = nil
	}

	if s.jobResultQueue != nil {
		s.jobResultQueue.Close()
		s.jobResultQueue = nil
	}

	if s.sourceQueue != nil {
		s.sourceQueue.Close()
		s.sourceQueue = nil
	}
}

func newDescribeSourceJob(a Source) DescribeSourceJob {
	daj := DescribeSourceJob{
		SourceID:             a.ID,
		DescribeResourceJobs: []DescribeResourceJob{},
		Status:               DescribeSourceJobCreated,
	}

	switch sType := SourceType(a.Type); sType {
	case SourceCloudAWS:
		for _, rType := range aws.ListResourceTypes() {
			daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
				ResourceType: rType,
				Status:       DescribeResourceJobCreated,
			})
		}
	case SourceCloudAzure:
		for _, rType := range azure.ListResourceTypes() {
			daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
				ResourceType: rType,
				Status:       DescribeResourceJobCreated,
			})
		}
	default:
		panic(fmt.Errorf("unsupported source type: %s", sType))
	}

	return daj
}

func enqueueDescribeResourceJobs(db Database, queue *Queue, a Source, daj DescribeSourceJob) {
	for i, drj := range daj.DescribeResourceJobs {
		nextStatus := DescribeResourceJobQueued
		errMsg := ""

		err := queue.PublishJSON("describe-scheduler", Job{
			JobID:               drj.ID,
			ParentJobID:         daj.ID,
			SourceType:          a.Type,
			ResourceType:        drj.ResourceType,
			DescribeCredentials: a.Credentials,
		})
		if err != nil {
			fmt.Printf("Failed to Queue DescribeResourceJob[%d]: %s\n", drj.ID, err.Error())

			nextStatus = DescribeResourceJobFailed
			errMsg = fmt.Sprintf("queue: %s", err.Error())
		}

		err = db.UpdateDescribeResourceJobStatus(drj.ID, nextStatus, errMsg)
		if err != nil {
			fmt.Printf("Failed to update DescribeResourceJob[%d]: %s\n", drj.ID, err.Error())
		}

		daj.DescribeResourceJobs[i].Status = nextStatus
	}
}
