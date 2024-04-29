package ingestion

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	logger *zap.Logger

	dataAgeRepo repo.DataAgeRepo

	ec2InstanceRepo   repo.EC2InstanceTypeRepo
	ebsVolumeTypeRepo repo.EBSVolumeTypeRepo
}

func New(logger *zap.Logger, ec2InstanceRepo repo.EC2InstanceTypeRepo, ebsVolumeRepo repo.EBSVolumeTypeRepo, dataAgeRepo repo.DataAgeRepo) *Service {
	return &Service{
		logger:            logger,
		ec2InstanceRepo:   ec2InstanceRepo,
		ebsVolumeTypeRepo: ebsVolumeRepo,
		dataAgeRepo:       dataAgeRepo,
	}
}

func (s *Service) Start(ctx context.Context) error {
	ticker := time.NewTimer(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		dataAge, err := s.dataAgeRepo.List()
		if err != nil {
			fmt.Println(err)
			continue
		}

		var ec2InstanceData *model.DataAge
		var ec2InstanceExtraData *model.DataAge
		for _, data := range dataAge {
			data := data
			switch data.DataType {
			case "AWS::EC2::Instance":
				ec2InstanceData = &data
			case "AWS::EC2::Instance::Extra":
				ec2InstanceExtraData = &data
			}
		}

		if ec2InstanceData == nil || ec2InstanceData.UpdatedAt.Before(time.Now().Add(-7*24*time.Hour)) {
			err = s.IngestEc2Instances()
			if err != nil {
				return err
			}
			if ec2InstanceData == nil {
				err = s.dataAgeRepo.Create(&model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					return err
				}
			} else {
				err = s.dataAgeRepo.Update("AWS::EC2::Instance", model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					return err
				}
			}
		}

		if ec2InstanceExtraData == nil || ec2InstanceExtraData.UpdatedAt.Before(time.Now().Add(-7*24*time.Hour)) {
			s.logger.Info("ingesting ec2 instance extra data")
			err = s.IngestEc2InstancesExtra(ctx)
			if err != nil {
				return err
			}
			if ec2InstanceExtraData == nil {
				err = s.dataAgeRepo.Create(&model.DataAge{
					DataType:  "AWS::EC2::Instance::Extra",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					return err
				}
			} else {
				err = s.dataAgeRepo.Update("AWS::EC2::Instance::Extra", model.DataAge{
					DataType:  "AWS::EC2::Instance::Extra",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Service) IngestEc2Instances() error {
	url := "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/index.csv"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	csvr := csv.NewReader(resp.Body)
	csvr.FieldsPerRecord = -1

	var columns map[string]int
	for {
		values, err := csvr.Read()
		if err != nil {
			return err
		}

		if len(values) > 2 {
			columns = readColumnPositions(values)
			break
		}
	}

	err = s.ec2InstanceRepo.Truncate()
	if err != nil {
		return err
	}
	// Read through each row in the CSV file and send a price.WithProduct on the results channel.
	for {
		row, err := csvr.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}

		switch row[columns["Product Family"]] {
		case "Compute Instance", "Compute Instance (bare metal)":
			v := model.EC2InstanceType{}
			v.PopulateFromMap(columns, row)

			if strings.ToLower(v.PhysicalProcessor) == "variable" {
				continue
			}
			if v.InstanceType == "" {
				continue
			}
			if v.TermType != "OnDemand" {
				continue
			}

			fmt.Println("Instance", v)
			err = s.ec2InstanceRepo.Create(&v)
			if err != nil {

				return err
			}
		case "Storage", "System Operation", "Provisioned Throughput":
			v := model.EBSVolumeType{}
			v.PopulateFromMap(columns, row)

			if v.VolumeType == "" {
				continue
			}
			if v.TermType != "OnDemand" {
				continue
			}
			fmt.Println("Volume", v)
			err = s.ebsVolumeTypeRepo.Create(&v)
			if err != nil {
				return err
			}
		}
	}
}

func (s *Service) IngestEc2InstancesExtra(ctx context.Context) error {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		s.logger.Error("failed to load SDK config", zap.Error(err))
		return err
	}
	if sdkConfig.Region == "" {
		sdkConfig.Region = "us-east-1"
	}
	baseEc2Client := ec2.NewFromConfig(sdkConfig)

	regions, err := baseEc2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{AllRegions: aws.Bool(true)})
	if err != nil {
		s.logger.Error("failed to describe regions", zap.Error(err))
		return err
	}
	for _, region := range regions.Regions {
		ec2Client := ec2.NewFromConfig(sdkConfig, func(o *ec2.Options) {
			o.Region = *region.RegionName
		})
		paginator := ec2.NewDescribeInstanceTypesPaginator(ec2Client, &ec2.DescribeInstanceTypesInput{})
		for paginator.HasMorePages() {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				s.logger.Error("failed to get next page", zap.Error(err), zap.String("region", *region.RegionName))
				return err
			}
			for _, instanceType := range output.InstanceTypes {
				extras := map[string]any{}
				if instanceType.EbsInfo != nil && instanceType.EbsInfo.EbsOptimizedInfo != nil {
					if instanceType.EbsInfo.EbsOptimizedInfo.BaselineBandwidthInMbps != nil {
						extras["ebs_baseline_bandwidth"] = *instanceType.EbsInfo.EbsOptimizedInfo.BaselineBandwidthInMbps
					}
					if instanceType.EbsInfo.EbsOptimizedInfo.MaximumBandwidthInMbps != nil {
						extras["ebs_maximum_bandwidth"] = *instanceType.EbsInfo.EbsOptimizedInfo.MaximumBandwidthInMbps
					}
					if instanceType.EbsInfo.EbsOptimizedInfo.BaselineIops != nil {
						extras["ebs_baseline_iops"] = *instanceType.EbsInfo.EbsOptimizedInfo.BaselineIops
					}
					if instanceType.EbsInfo.EbsOptimizedInfo.MaximumIops != nil {
						extras["ebs_maximum_iops"] = *instanceType.EbsInfo.EbsOptimizedInfo.MaximumIops
					}
					if instanceType.EbsInfo.EbsOptimizedInfo.BaselineThroughputInMBps != nil {
						extras["ebs_baseline_throughput"] = *instanceType.EbsInfo.EbsOptimizedInfo.BaselineThroughputInMBps
					}
					if instanceType.EbsInfo.EbsOptimizedInfo.MaximumThroughputInMBps != nil {
						extras["ebs_maximum_throughput"] = *instanceType.EbsInfo.EbsOptimizedInfo.MaximumThroughputInMBps
					}
				}
				if len(extras) == 0 {
					s.logger.Warn("no extras found", zap.String("region", *region.RegionName), zap.String("instanceType", string(instanceType.InstanceType)))
					continue
				}
				s.logger.Info("updating extras", zap.String("region", *region.RegionName), zap.String("instanceType", string(instanceType.InstanceType)), zap.Any("extras", extras))
				err = s.ec2InstanceRepo.UpdateExtrasByRegionAndType(*region.RegionName, string(instanceType.InstanceType), extras)
				if err != nil {
					s.logger.Error("failed to update extras", zap.Error(err), zap.String("region", *region.RegionName), zap.String("instanceType", string(instanceType.InstanceType)))
					return err
				}
			}
		}
	}
	return nil
}

// readColumnPositions maps column names to their position in the CSV file.
func readColumnPositions(values []string) map[string]int {
	columns := make(map[string]int)
	for i, v := range values {
		columns[v] = i
	}
	return columns
}
