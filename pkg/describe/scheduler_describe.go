package describe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"go.uber.org/zap"
)

func (s Scheduler) RunDescribeJobScheduler() {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleDescribeJob()
	}
}

func (s Scheduler) scheduleDescribeJob() {
	s.logger.Info("scheduleDescribeJob")
	err := s.CheckWorkspaceResourceLimit()
	if err != nil {
		s.logger.Error("failure on CheckWorkspaceResourceLimit", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	connections, err := s.db.ListSources()
	if err != nil {
		s.logger.Error("Failed to fetch all connections", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}
	for _, connection := range connections {
		err = s.describeConnection(connection, true)
		if err != nil {
			s.logger.Error("Failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.Error(err))
			DescribeSourceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		DescribeSourceJobsCount.WithLabelValues("successful").Inc()
	}
	DescribeJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) describeConnection(connection Source, scheduled bool) error {
	healthCheckedSrc, err := s.onboardClient.GetSourceHealthcheck(&httpclient.Context{
		UserRole: api2.EditorRole,
	}, connection.ID.String())
	if err != nil {
		return err
	}

	if scheduled && healthCheckedSrc.AssetDiscoveryMethod != source.AssetDiscoveryMethodTypeScheduled {
		return errors.New("asset discovery is not scheduled")
	}
	if healthCheckedSrc.HealthState == source.HealthStatusUnhealthy {
		return errors.New("connection is not healthy")
	}

	job, err := s.db.GetLastDescribeSourceJob(connection.ID)
	if err != nil {
		return err
	}

	if !scheduled || job == nil || job.UpdatedAt.Before(time.Now().Add(time.Duration(-s.describeIntervalHours)*time.Hour)) {
		err = s.createCloudNativeDescribeSource(&connection, job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Scheduler) createCloudNativeDescribeSource(source *Source, src *DescribeSourceJob) error {
	describedAt := time.Now()
	triggerType := enums.DescribeTriggerTypeScheduled
	if src == nil {
		triggerType = enums.DescribeTriggerTypeInitialDiscovery
	}

	s.logger.Info("Source is due for a describe. Creating a job now", zap.String("sourceId", source.ID.String()))

	daj := newDescribeSourceJob(*source, describedAt)
	err := s.db.CreateDescribeSourceJob(&daj)
	if err != nil {
		return err
	}
	describeSourceJobFailure := false
	defer func() {
		if describeSourceJobFailure == true {
			if err = s.db.UpdateDescribeSourceJob(daj.ID, api.DescribeSourceJobCompletedWithFailure); err != nil {
				s.logger.Error("failed to update UpdateDescribeSourceJob in failure", zap.Error(err), zap.Uint("parentJobId", daj.ID))
			}
			if err = s.db.UpdateDescribeResourceJobStatusByParentId(daj.ID, api.DescribeResourceJobFailed, ""); err != nil {
				s.logger.Error("Failed to update DescribeResourceJob in failure", zap.Error(err), zap.Uint("parentJobId", daj.ID))
			}
		}
	}()

	cloudDaj, err := newCloudNativeDescribeSourceJob(daj)
	if err != nil {
		describeSourceJobFailure = true
		return err
	}

	err = s.db.CreateCloudNativeDescribeSourceJob(&cloudDaj)
	if err != nil {
		describeSourceJobFailure = true
		return err
	}

	err = enqueueCloudNativeDescribeConnectionJob(s.logger, s.db, CurrentWorkspaceID, s.cloudNativeAPIBaseURL,
		s.cloudNativeAPIAuthKey, *source, cloudDaj, s.kafkaResourcesTopic, describedAt, triggerType)
	if err != nil {
		describeSourceJobFailure = true
		return err
	}

	errUpdate := s.db.UpdateDescribeSourceJob(daj.ID, api.DescribeSourceJobInProgress)
	err = s.db.UpdateSourceDescribed(source.ID, describedAt, time.Duration(s.describeIntervalHours)*time.Hour)

	if errUpdate != nil {
		return errUpdate
	}
	return err
}

func newDescribeSourceJob(a Source, describedAt time.Time) DescribeSourceJob {
	daj := DescribeSourceJob{
		DescribedAt:          describedAt,
		SourceID:             a.ID,
		AccountID:            a.AccountID,
		DescribeResourceJobs: []DescribeResourceJob{},
		Status:               api.DescribeSourceJobCreated,
	}
	var resourceTypes []string
	switch a.Type {
	case api.SourceCloudAWS:
		resourceTypes = aws.ListResourceTypes()
	case api.SourceCloudAzure:
		resourceTypes = azure.ListResourceTypes()
	default:
		panic(fmt.Errorf("unsupported source type: %s", a.Type))
	}
	rand.Shuffle(len(resourceTypes), func(i, j int) { resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i] })
	for _, rType := range resourceTypes {
		daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
			ResourceType: rType,
			Status:       api.DescribeResourceJobCreated,
		})
	}
	return daj
}

func newCloudNativeDescribeSourceJob(j DescribeSourceJob) (CloudNativeDescribeSourceJob, error) {
	credentialsKeypair, err := crypto.GenerateKey(j.AccountID, j.SourceID.String(), "x25519", 0)
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	credentialsPrivateKey, err := credentialsKeypair.Armor()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	credentialsPublicKey, err := credentialsKeypair.GetArmoredPublicKey()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}

	resultEncryptionKeyPair, err := crypto.GenerateKey(j.AccountID, j.SourceID.String(), "x25519", 0)
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	resultEncryptionPrivateKey, err := resultEncryptionKeyPair.Armor()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	resultEncryptionPublicKey, err := resultEncryptionKeyPair.GetArmoredPublicKey()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}

	job := CloudNativeDescribeSourceJob{
		SourceJob:                      j,
		CredentialEncryptionPrivateKey: credentialsPrivateKey,
		CredentialEncryptionPublicKey:  credentialsPublicKey,
		ResultEncryptionPrivateKey:     resultEncryptionPrivateKey,
		ResultEncryptionPublicKey:      resultEncryptionPublicKey,
	}

	return job, nil
}

func enqueueCloudNativeDescribeConnectionJob(logger *zap.Logger, db Database, workspaceId string,
	cloudNativeAPIBaseURL string, cloudNativeAPIAuthKey string, a Source, daj CloudNativeDescribeSourceJob,
	kafkaResourcesTopic string, describedAt time.Time, triggerType enums.DescribeTriggerType) error {

	resourceJobs := map[uint]string{}
	for _, drj := range daj.SourceJob.DescribeResourceJobs {
		resourceJobs[drj.ID] = drj.ResourceType
	}
	dcj := DescribeConnectionJob{
		JobID:        daj.SourceJob.ID,
		ResourceJobs: resourceJobs,
		SourceID:     daj.SourceJob.SourceID.String(),
		AccountID:    daj.SourceJob.AccountID,
		DescribedAt:  describedAt.UnixMilli(),
		SourceType:   a.Type,
		ConfigReg:    a.ConfigRef,
		TriggerType:  triggerType,
	}
	dcjJson, err := json.Marshal(dcj)
	if err != nil {
		return fmt.Errorf("failed to marshal DescribeConnectionJob due to %v", err)
	}

	cloudTriggerInput := api.CloudNativeConnectionWorkerTriggerInput{
		WorkspaceID:             workspaceId,
		JobID:                   daj.JobID.String(),
		JobJson:                 string(dcjJson),
		CredentialsCallbackURL:  fmt.Sprintf("%s/schedule/api/v1/jobs/%s/creds", IngressBaseURL, daj.JobID.String()),
		EndOfJobCallbackURL:     fmt.Sprintf("%s/schedule/api/v1/jobs/%s/callback", IngressBaseURL, daj.JobID.String()),
		CredentialDecryptionKey: daj.CredentialEncryptionPrivateKey,
		OutputEncryptionKey:     daj.ResultEncryptionPublicKey,
		ResourcesTopic:          kafkaResourcesTopic,
	}

	//call azure function to trigger describe connection job
	cloudTriggerInputJson, err := json.Marshal(cloudTriggerInput)
	if err != nil {
		return fmt.Errorf("failed to marshal cloudTriggerInput due to %v", err)
	}
	//enqueue job to cloud native connection worker
	httpClient := &http.Client{
		Timeout: 5 * time.Minute,
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/orchestrators/ConnectionWorkerOrchestrator", cloudNativeAPIBaseURL), bytes.NewBuffer(cloudTriggerInputJson))
	if err != nil {
		return fmt.Errorf("failed to create http request due to %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-kaytu-cloud-auth-key", cloudNativeAPIAuthKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send orchestrators http request due to %v", err)
	}

	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read orchestrators http response due to %v", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to trigger cloud native connection worker due to %s", string(resBody))
	}

	if err := db.UpdateDescribeResourceJobStatusByParentId(daj.SourceJob.ID, api.DescribeResourceJobQueued, ""); err != nil {
		logger.Error("Failed to update DescribeResourceJob",
			zap.Uint("parentJobId", daj.ID),
			zap.Error(err),
		)
	}
	for i := range daj.SourceJob.DescribeResourceJobs {
		daj.SourceJob.DescribeResourceJobs[i].Status = api.DescribeResourceJobQueued
	}
	return nil
}
