package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db/model"
	"go.uber.org/zap"
	"time"
)

type FirehoseMeter struct {
	WorkspaceID string             `json:"workspaceId"`
	UsageDate   time.Time          `json:"usageDate"`
	MeterType   entities.MeterType `json:"meterType"`
	CreatedAt   time.Time          `json:"createdAt"`
	Value       int64              `json:"value"`
}

func (svc MeteringService) sendMeterToFirehose(ctx context.Context, meter *model.Meter) error {
	fhMeter := FirehoseMeter{
		WorkspaceID: meter.WorkspaceID,
		UsageDate:   meter.UsageDate,
		MeterType:   meter.MeterType,
		CreatedAt:   meter.CreatedAt,
		Value:       meter.Value,
	}
	jsonFhMeter, err := json.Marshal(fhMeter)
	if err != nil {
		svc.logger.Error("failed to marshal meter", zap.Error(err))
	}

	base64EncodedFhMeter := []byte(base64.StdEncoding.EncodeToString(jsonFhMeter))

	_, err = svc.firehoseClient.PutRecord(ctx, &firehose.PutRecordInput{
		DeliveryStreamName: &svc.cnf.UsageMetersFirehoseStreamName,
		Record: &types.Record{
			Data: base64EncodedFhMeter,
		},
	})
	if err != nil {
		svc.logger.Error("failed to send meter to firehose", zap.Error(err))
		return err
	}

	return nil
}

func (svc MeteringService) sendMetersToFirehose(ctx context.Context, meters []*model.Meter) error {
	var firehoseRecords []types.Record
	for _, meter := range meters {
		fhMeter := FirehoseMeter{
			WorkspaceID: meter.WorkspaceID,
			UsageDate:   meter.UsageDate,
			MeterType:   meter.MeterType,
			CreatedAt:   meter.CreatedAt,
			Value:       meter.Value,
		}
		jsonFhMeter, err := json.Marshal(fhMeter)
		if err != nil {
			svc.logger.Error("failed to marshal meter", zap.Error(err))
		}

		base64EncodedFhMeter := []byte(base64.StdEncoding.EncodeToString(jsonFhMeter))

		firehoseRecords = append(firehoseRecords, types.Record{
			Data: base64EncodedFhMeter,
		})
	}

	_, err := svc.firehoseClient.PutRecordBatch(ctx, &firehose.PutRecordBatchInput{
		DeliveryStreamName: &svc.cnf.UsageMetersFirehoseStreamName,
		Records:            firehoseRecords,
	})
	if err != nil {
		svc.logger.Error("failed to send meter to firehose", zap.Error(err))
		return err
	}

	return nil
}
