package compliancebuilder

import (
	es2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"

	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type alarmsBuilder struct {
	client          keibi.Client
	summarizerJobID uint
	alarms          []es.FindingAlarm
}

func NewAlarmsBuilder(client keibi.Client, summarizerJobID uint) *alarmsBuilder {
	return &alarmsBuilder{
		client:          client,
		summarizerJobID: summarizerJobID,
		alarms:          nil,
	}
}

func (b *alarmsBuilder) Process(resource es2.Finding) error {
	activeAlarm, err := query.GetLastActiveAlarm(b.client, resource.ResourceID, resource.PolicyID)
	if err != nil {
		return err
	}

	if activeAlarm == nil { // there's no alarm
		if resource.Status != types.ComplianceResultOK {
			// create a new one
			b.alarms = append(b.alarms, es.FindingAlarm{
				ResourceID:     resource.ResourceID,
				BenchmarkID:    resource.BenchmarkID,
				ControlID:      resource.PolicyID,
				ResourceType:   resource.ResourceType,
				ServiceName:    resource.ServiceName,
				SourceID:       resource.ConnectionID,
				SourceType:     resource.Connector,
				PolicySeverity: resource.PolicySeverity,
				CreatedAt:      resource.DescribedAt,
				ScheduleJobID:  resource.ScheduleJobID,
				LastEvaluated:  resource.DescribedAt,
				Status:         resource.Status,
				Events: []es.Event{
					{
						ResourceID:    resource.ResourceID,
						ControlID:     resource.PolicyID,
						CreatedAt:     resource.DescribedAt,
						ScheduleJobID: resource.ScheduleJobID,
						Status:        resource.Status,
					},
				},
			})
		} else {
			// it's ok and there's no active alarm
			// do nothing
		}
	} else {
		// add finding to events, update the alarm
		activeAlarm.Status = resource.Status
		activeAlarm.LastEvaluated = resource.DescribedAt
		activeAlarm.Events = append(activeAlarm.Events, es.Event{
			ResourceID:    resource.ResourceID,
			ControlID:     resource.PolicyID,
			CreatedAt:     resource.DescribedAt,
			ScheduleJobID: resource.ScheduleJobID,
			Status:        resource.Status,
		})
		b.alarms = append(b.alarms, *activeAlarm)
	}
	return nil
}

func (b *alarmsBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *alarmsBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.alarms {
		docs = append(docs, v)
	}
	return docs
}

func (b *alarmsBuilder) Cleanup(scheduleJobID uint) error {
	return nil
}
