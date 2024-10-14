package ingestion

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/opengovern/opengovernance/services/wastage/db/connector"
	"github.com/opengovern/opengovernance/services/wastage/db/model"
	"github.com/opengovern/opengovernance/services/wastage/db/repo"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	logger *zap.Logger

	DataAgeRepo repo.DataAgeRepo

	db                *connector.Database
	ec2InstanceRepo   repo.EC2InstanceTypeRepo
	rdsRepo           repo.RDSProductRepo
	rdsInstanceRepo   repo.RDSDBInstanceRepo
	ebsVolumeTypeRepo repo.EBSVolumeTypeRepo
	storageRepo       repo.RDSDBStorageRepo
}

func New(logger *zap.Logger, db *connector.Database, ec2InstanceRepo repo.EC2InstanceTypeRepo, rdsRepo repo.RDSProductRepo, rdsInstanceRepo repo.RDSDBInstanceRepo, storageRepo repo.RDSDBStorageRepo, ebsVolumeRepo repo.EBSVolumeTypeRepo, dataAgeRepo repo.DataAgeRepo) *Service {
	return &Service{
		logger:            logger,
		db:                db,
		ec2InstanceRepo:   ec2InstanceRepo,
		rdsInstanceRepo:   rdsInstanceRepo,
		rdsRepo:           rdsRepo,
		storageRepo:       storageRepo,
		ebsVolumeTypeRepo: ebsVolumeRepo,
		DataAgeRepo:       dataAgeRepo,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.logger.Info("Ingestion service started")
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("ingestion paniced", zap.Error(fmt.Errorf("%v", r)))
			time.Sleep(15 * time.Minute)
			go s.Start(ctx)
		}
	}()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.logger.Info("checking data age")
		dataAge, err := s.DataAgeRepo.List()
		if err != nil {
			s.logger.Error("failed to list data age", zap.Error(err))
			continue
		}

		var ec2InstanceData *model.DataAge
		var rdsData *model.DataAge
		for _, data := range dataAge {
			data := data
			switch data.DataType {
			case "AWS::EC2::Instance":
				ec2InstanceData = &data
			case "AWS::RDS::Instance":
				rdsData = &data
			}
		}

		if ec2InstanceData == nil || ec2InstanceData.UpdatedAt.Before(time.Now().Add(-365*24*time.Hour)) {
			s.logger.Info("ec2 instance ingest started")
			err = s.IngestEc2Instances(ctx)
			if err != nil {
				s.logger.Error("failed to ingest ec2 instances", zap.Error(err))
				continue
			}
			if ec2InstanceData == nil {
				err = s.DataAgeRepo.Create(&model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error("failed to create data age", zap.Error(err))
					continue
				}
			} else {
				err = s.DataAgeRepo.Update("AWS::EC2::Instance", model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error("failed to update data age", zap.Error(err))
					continue
				}
			}
		} else {
			s.logger.Info("ec2 instance ingest not started: ", zap.Any("usage", ec2InstanceData))
		}

		if rdsData == nil || rdsData.UpdatedAt.Before(time.Now().Add(-7*24*time.Hour)) {
			s.logger.Info("rds ingest started")
			err = s.IngestRDS()
			if err != nil {
				s.logger.Error("failed to ingest rds", zap.Error(err))
				continue
			}
			if rdsData == nil {
				err = s.DataAgeRepo.Create(&model.DataAge{
					DataType:  "AWS::RDS::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error("failed to create rds data age", zap.Error(err))
					continue
				}
			} else {
				err = s.DataAgeRepo.Update("AWS::RDS::Instance", model.DataAge{
					DataType:  "AWS::RDS::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error("failed to update rds data age", zap.Error(err))
					continue
				}
			}
		} else {
			s.logger.Info("rds ingest not started: ", zap.Any("usage", rdsData))
		}
	}

	s.logger.Error("Ingestion service stopped", zap.Time("time", time.Now()))
}

func (s *Service) IngestEc2Instances(ctx context.Context) error {
	//transaction := s.db.Conn().Begin()
	//defer func() {
	//	transaction.Rollback()
	//}()
	ec2InstanceTypeTable, err := s.ec2InstanceRepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "ec2_instance_type"),
			zap.Error(err))
		return err
	}

	ebsVolumeTypeTable, err := s.ebsVolumeTypeRepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "ebs_volume_type"),
			zap.Error(err))
		return err
	}
	err = s.ingestEc2InstancesBase(ctx, ec2InstanceTypeTable, ebsVolumeTypeTable, nil)
	if err != nil {
		s.logger.Error("failed to ingest ec2 instances", zap.Error(err))
		return err
	}

	err = s.ingestEc2InstancesExtra(ctx, ec2InstanceTypeTable, nil)
	if err != nil {
		s.logger.Error("failed to ingest ec2 instances extra", zap.Error(err))
		return err
	}

	//err = transaction.Commit().Error
	//if err != nil {
	//	s.logger.Error("failed to commit transaction", zap.Error(err))
	//	return err
	//}

	s.logger.Info("ingested ec2 instances")

	return nil
}

func (s *Service) ingestEc2InstancesBase(ctx context.Context, ec2InstanceTypeTable, ebsVolumeTypeTable string, transaction *gorm.DB) error {
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

	// Read through each row in the CSV file and send a price.WithProduct on the results channel.
	for {
		row, err := csvr.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
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
			err = s.ec2InstanceRepo.Create(ec2InstanceTypeTable, transaction, &v)
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
			err = s.ebsVolumeTypeRepo.Create(ebsVolumeTypeTable, transaction, &v)
			if err != nil {
				return err
			}
		}
	}

	err = s.ec2InstanceRepo.MoveViewTransaction(ec2InstanceTypeTable)
	if err != nil {
		return err
	}

	err = s.ebsVolumeTypeRepo.MoveViewTransaction(ebsVolumeTypeTable)
	if err != nil {
		return err
	}

	err = s.ec2InstanceRepo.RemoveOldTables(ec2InstanceTypeTable)
	if err != nil {
		return err
	}

	err = s.ebsVolumeTypeRepo.RemoveOldTables(ebsVolumeTypeTable)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ingestEc2InstancesExtra(ctx context.Context, ec2InstanceTypeTable string, transaction *gorm.DB) error {
	sdkConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		s.logger.Error("failed to load SDK config", zap.Error(err))
		return err
	}
	baseEc2Client := ec2.NewFromConfig(sdkConfig)

	regions, err := baseEc2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{AllRegions: aws.Bool(false)})
	if err != nil {
		s.logger.Error("failed to describe regions", zap.Error(err))
		return err
	}

	for _, region := range regions.Regions {
		cnf, err := config.LoadDefaultConfig(ctx, config.WithRegion(*region.RegionName))
		if err != nil {
			s.logger.Error("failed to load SDK config", zap.Error(err), zap.String("region", *region.RegionName))
			return err
		}
		ec2Client := ec2.NewFromConfig(cnf)
		paginator := ec2.NewDescribeInstanceTypesPaginator(ec2Client, &ec2.DescribeInstanceTypesInput{})
		for paginator.HasMorePages() {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				s.logger.Error("failed to get next page", zap.Error(err), zap.String("region", *region.RegionName))
				return err
			}
			for _, instanceType := range output.InstanceTypes {
				extras := getEc2InstanceExtrasMap(instanceType)
				if len(extras) == 0 {
					s.logger.Warn("no extras found", zap.String("region", *region.RegionName), zap.String("instanceType", string(instanceType.InstanceType)))
					continue
				}
				s.logger.Info("updating extras", zap.String("region", *region.RegionName), zap.String("instanceType", string(instanceType.InstanceType)), zap.Any("extras", extras))
				err = s.ec2InstanceRepo.UpdateExtrasByRegionAndType(ec2InstanceTypeTable, transaction, *region.RegionName, string(instanceType.InstanceType), extras)
				if err != nil {
					s.logger.Error("failed to update extras", zap.Error(err), zap.String("region", *region.RegionName), zap.String("instanceType", string(instanceType.InstanceType)))
					return err
				}
			}
		}
	}

	// Populate the still missing extras with the us-east-1 region data
	paginator := ec2.NewDescribeInstanceTypesPaginator(baseEc2Client, &ec2.DescribeInstanceTypesInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.Error("failed to get next page", zap.Error(err), zap.String("region", "all"))
			return err
		}
		for _, instanceType := range output.InstanceTypes {
			extras := getEc2InstanceExtrasMap(instanceType)
			if len(extras) == 0 {
				s.logger.Warn("no extras found", zap.String("region", "all"), zap.String("instanceType", string(instanceType.InstanceType)))
				continue
			}
			s.logger.Info("updating extras", zap.String("region", "all"), zap.String("instanceType", string(instanceType.InstanceType)), zap.Any("extras", extras))
			err = s.ec2InstanceRepo.UpdateNullExtrasByType(ec2InstanceTypeTable, transaction, string(instanceType.InstanceType), extras)
			if err != nil {
				s.logger.Error("failed to update extras", zap.Error(err), zap.String("region", "all"), zap.String("instanceType", string(instanceType.InstanceType)))
				return err
			}
		}
	}

	return nil
}

func (s *Service) IngestRDS() error {
	rdsInstancesTable, err := s.rdsInstanceRepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "rdsdb_instances"),
			zap.Error(err))
		return err
	}
	rdsStorageTable, err := s.storageRepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "rdsdb_storages"),
			zap.Error(err))
		return err
	}
	rdsProductsTable, err := s.rdsRepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "rds_products"),
			zap.Error(err))
		return err
	}

	url := "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonRDS/current/index.csv"
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
	//
	//transaction := s.db.Conn().Begin()
	//defer func() {
	//	transaction.Rollback()
	//}()

	var transaction *gorm.DB

	// Read through each row in the CSV file and send a price.WithProduct on the results channel.
	for {
		row, err := csvr.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		switch row[columns["Product Family"]] {
		case "Database Storage", "Provisioned IOPS", "Provisioned Throughput", "System Operation":
			v := model.RDSDBStorage{}
			v.PopulateFromMap(columns, row)

			if !v.DoIngest() {
				continue
			}

			fmt.Println("RDSDBStorage", v)

			err = s.storageRepo.Create(rdsStorageTable, transaction, &v)
			if err != nil {
				return err
			}

		case "Database Instance":
			v := model.RDSDBInstance{}
			v.PopulateFromMap(columns, row)

			if v.TermType != "OnDemand" {
				continue
			}
			if v.LocationType == "AWS Outposts" {
				continue
			}

			fmt.Println("RDSDBInstance", v)

			err = s.rdsInstanceRepo.Create(rdsInstancesTable, transaction, &v)
			if err != nil {
				return err
			}

		default:
			v := model.RDSProduct{}
			v.PopulateFromMap(columns, row)

			if v.TermType != "OnDemand" {
				continue
			}
			if v.LocationType == "AWS Outposts" {
				continue
			}

			fmt.Println("RDS", v)

			err = s.rdsRepo.Create(rdsProductsTable, transaction, &v)
			if err != nil {
				return err
			}
		}
	}

	err = s.rdsInstanceRepo.UpdateNilEBSThroughput(transaction, rdsInstancesTable)
	if err != nil {
		s.logger.Error("failed to update nil ebs throughput", zap.Error(err))
	}

	err = s.rdsInstanceRepo.MoveViewTransaction(rdsInstancesTable)
	if err != nil {
		s.logger.Error("failed to move view", zap.String("table", rdsInstancesTable), zap.Error(err))
		return err
	}

	err = s.rdsRepo.MoveViewTransaction(rdsProductsTable)
	if err != nil {
		s.logger.Error("failed to move view", zap.String("table", rdsProductsTable), zap.Error(err))
		return err
	}

	err = s.storageRepo.MoveViewTransaction(rdsStorageTable)
	if err != nil {
		s.logger.Error("failed to move view", zap.String("table", rdsStorageTable), zap.Error(err))
		return err
	}

	err = s.rdsInstanceRepo.RemoveOldTables(rdsInstancesTable)
	if err != nil {
		s.logger.Error("failed to remove old tables", zap.String("table", rdsInstancesTable), zap.Error(err))
		return err
	}

	err = s.rdsRepo.RemoveOldTables(rdsProductsTable)
	if err != nil {
		s.logger.Error("failed to remove old tables", zap.String("table", rdsProductsTable), zap.Error(err))
		return err
	}

	err = s.storageRepo.RemoveOldTables(rdsStorageTable)
	if err != nil {
		s.logger.Error("failed to remove old tables", zap.String("table", rdsStorageTable), zap.Error(err))
		return err
	}

	//err = transaction.Commit().Error
	//if err != nil {
	//	return err
	//}
	return nil
}

func getEc2InstanceExtrasMap(instanceType ec2types.InstanceTypeInfo) map[string]any {
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
	return extras
}

// readColumnPositions maps column names to their position in the CSV file.
func readColumnPositions(values []string) map[string]int {
	columns := make(map[string]int)
	for i, v := range values {
		columns[v] = i
	}
	return columns
}
