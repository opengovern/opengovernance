package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"go.uber.org/zap"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"gorm.io/gorm"
	"strings"
	"time"
)

var (
	services = map[string]string{
		"ComputeEngine": "6F81-5844-456A",
	}
)

const (
	ram = "RAM"
	cpu = "CPU"
)

type GcpService struct {
	logger *zap.Logger

	apiService *cloudbilling.APIService
	compute    *compute.Service
	project    string

	DataAgeRepo repo.DataAgeRepo

	db                     *connector.Database
	computeMachineTypeRepo repo.GCPComputeMachineTypeRepo
	computeDiskTypeRepo    repo.GCPComputeDiskTypeRepo
	computeSKURepo         repo.GCPComputeSKURepo
}

func NewGcpService(ctx context.Context, logger *zap.Logger, dataAgeRepo repo.DataAgeRepo, computeMachineTypeRepo repo.GCPComputeMachineTypeRepo,
	computeStorageTypeRepo repo.GCPComputeDiskTypeRepo, computeSKURepo repo.GCPComputeSKURepo, db *connector.Database, gcpCredentials map[string]string, projectId string) (*GcpService, error) {
	configJson, err := json.Marshal(gcpCredentials)
	if err != nil {
		return nil, err
	}
	gcpOpts := []option.ClientOption{
		option.WithCredentialsJSON(configJson),
	}
	apiService, err := cloudbilling.NewService(ctx, gcpOpts...)
	if err != nil {
		return nil, err
	}

	compute, err := compute.NewService(ctx, gcpOpts...)
	if err != nil {
		return nil, err
	}

	return &GcpService{
		logger:                 logger,
		DataAgeRepo:            dataAgeRepo,
		db:                     db,
		apiService:             apiService,
		compute:                compute,
		computeSKURepo:         computeSKURepo,
		computeDiskTypeRepo:    computeStorageTypeRepo,
		computeMachineTypeRepo: computeMachineTypeRepo,
		project:                projectId,
	}, nil
}

func (s *GcpService) Start(ctx context.Context) {
	s.logger.Info("GCP Ingestion service started")
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("gcp ingestion paniced", zap.Error(fmt.Errorf("%v", r)))
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

		var computeData *model.DataAge
		for _, data := range dataAge {
			data := data
			switch data.DataType {
			case "GCPComputeEngine":
				computeData = &data
			}
		}

		if computeData == nil || computeData.UpdatedAt.Before(time.Now().Add(-365*24*time.Hour)) {
			s.logger.Info("gcp compute engine ingest started")
			err = s.IngestComputeInstance(ctx)
			if err != nil {
				s.logger.Error("failed to ingest gcp compute engine", zap.Error(err))
				continue
			}
			if computeData == nil {
				err = s.DataAgeRepo.Create(&model.DataAge{
					DataType:  "GCPComputeEngine",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error("failed to create data age", zap.Error(err))
					continue
				}
			} else {
				err = s.DataAgeRepo.Update("GCPComputeEngine", model.DataAge{
					DataType:  "GCPComputeEngine",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error("failed to update data age", zap.Error(err))
					continue
				}
			}
		} else {
			s.logger.Info("gcp compute engine ingest not started: ", zap.Any("usage", computeData))
		}
	}
}

func (s *GcpService) IngestComputeInstance(ctx context.Context) error {
	computeMachineTypeTable, err := s.computeMachineTypeRepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "compute_machine_type"),
			zap.Error(err))
		return err
	}

	computeDiskTable, err := s.computeDiskTypeRepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "compute_disk_type"),
			zap.Error(err))
		return err
	}

	computeSKUTable, err := s.computeSKURepo.CreateNewTable()
	if err != nil {
		s.logger.Error("failed to auto migrate",
			zap.String("table", "compute_sku"),
			zap.Error(err))
		return err
	}

	var transaction *gorm.DB
	machineTypePrices := make(map[string]map[string]float64)
	storageTypePrices := make(map[string]map[string]float64)
	skus, err := s.fetchSKUs(ctx, services["ComputeEngine"])
	if err != nil {
		return err
	}
	for _, sku := range skus {
		if sku.PricingInfo == nil || len(sku.PricingInfo) == 0 || sku.PricingInfo[len(sku.PricingInfo)-1].PricingExpression == nil {
			continue
		}
		if len(sku.PricingInfo[len(sku.PricingInfo)-1].PricingExpression.TieredRates) == 0 {
			continue
		}

		mf, rg, t := model.GetSkuDetails(sku)
		if rg == cpu || rg == ram {
			if _, ok := machineTypePrices[fmt.Sprintf("%s.%s", mf, rg)]; !ok {
				skuMachineTypePrices := make(map[string]float64)
				machineTypePrices[fmt.Sprintf("%s.%s", mf, rg)] = skuMachineTypePrices
			}
		}
		if rg == "SSD" && strings.Contains(sku.Description, "Hyperdisk Throughput Capacity") {
			skuStorageTypePrices := make(map[string]float64)
			storageTypePrices["hyperdisk-throughput"] = skuStorageTypePrices
		}
		if rg == "SSD" && strings.Contains(sku.Description, "Hyperdisk Extreme Capacity") {
			skuStorageTypePrices := make(map[string]float64)
			storageTypePrices["hyperdisk-extreme"] = skuStorageTypePrices
		}
		if rg == "SSD" && strings.Contains(sku.Description, "Hyperdisk Balanced Capacity") {
			skuStorageTypePrices := make(map[string]float64)
			storageTypePrices["hyperdisk-balanced"] = skuStorageTypePrices
		}
		if sku.Description == "Storage PD Capacity" {
			skuStorageTypePrices := make(map[string]float64)
			storageTypePrices["pd-standard"] = skuStorageTypePrices
		}
		if sku.Description == "Balanced PD Capacity" {
			skuStorageTypePrices := make(map[string]float64)
			storageTypePrices["pd-balanced"] = skuStorageTypePrices
		}
		if sku.Description == "SSD backed PD Capacity" {
			skuStorageTypePrices := make(map[string]float64)
			storageTypePrices["pd-ssd"] = skuStorageTypePrices
		}
		if sku.Description == "Extreme PD Capacity" {
			skuStorageTypePrices := make(map[string]float64)
			storageTypePrices["pd-extreme"] = skuStorageTypePrices
		}
		for _, region := range sku.ServiceRegions {
			computeSKU := &model.GCPComputeSKU{}
			computeSKU.PopulateFromObject(sku, region)

			err = s.computeSKURepo.Create(computeSKUTable, transaction, computeSKU)
			if err != nil {
				return err
			}

			if (rg == cpu || rg == ram) && t == "Predefined" {
				machineTypePrices[fmt.Sprintf("%s.%s", mf, rg)][region] = computeSKU.UnitPrice
			}
			if computeSKU.ResourceGroup == "SSD" && strings.Contains(computeSKU.Description, "Hyperdisk Throughput Capacity") {
				storageTypePrices["hyperdisk-throughput"][region] = computeSKU.UnitPrice
			}
			if computeSKU.ResourceGroup == "SSD" && strings.Contains(computeSKU.Description, "Hyperdisk Extreme Capacity") {
				storageTypePrices["hyperdisk-extreme"][region] = computeSKU.UnitPrice
			}
			if computeSKU.ResourceGroup == "SSD" && strings.Contains(computeSKU.Description, "Hyperdisk Balanced Capacity") {
				storageTypePrices["hyperdisk-balanced"][region] = computeSKU.UnitPrice
			}
			if computeSKU.Description == "Storage PD Capacity" {
				storageTypePrices["pd-standard"][region] = computeSKU.UnitPrice
			}
			if computeSKU.Description == "Balanced PD Capacity" {
				storageTypePrices["pd-balanced"][region] = computeSKU.UnitPrice
			}
			if computeSKU.Description == "SSD backed PD Capacity" {
				storageTypePrices["pd-ssd"][region] = computeSKU.UnitPrice
			}
			if computeSKU.Description == "Extreme PD Capacity" {
				storageTypePrices["pd-extreme"][region] = computeSKU.UnitPrice
			}
		}
	}

	types, err := s.fetchMachineTypes(ctx)
	if err != nil {
		s.logger.Error("failed to fetch machine types", zap.Error(err))
		return err
	}
	s.logger.Info("fetched machine types", zap.Any("count", len(types)))
	for _, mt := range types {
		computeMachineType := &model.GCPComputeMachineType{}
		computeMachineType.PopulateFromObject(mt)

		region := strings.Join([]string{strings.Split(mt.Zone, "-")[0], strings.Split(mt.Zone, "-")[1]}, "-")
		computeMachineType.Region = region

		mf := strings.ToLower(strings.Split(mt.Name, "-")[0])
		rp, ok := machineTypePrices[fmt.Sprintf("%s.%s", mf, ram)][region]
		if !ok {
			s.logger.Error("failed to get ram price", zap.String("machine_type", mt.Name), zap.String("family", fmt.Sprintf("%s.%s", mf, cpu)),
				zap.String("region", region), zap.Any("prices", machineTypePrices[fmt.Sprintf("%s.%s", mf, ram)]))
			continue
		}

		cp, ok := machineTypePrices[fmt.Sprintf("%s.%s", mf, cpu)][region]
		if !ok {
			s.logger.Error("failed to get cpu price", zap.String("machine_type", mt.Name), zap.String("family", fmt.Sprintf("%s.%s", mf, cpu)),
				zap.String("region", region), zap.Any("prices", machineTypePrices[fmt.Sprintf("%s.%s", mf, cpu)]))
			continue
		}

		computeMachineType.UnitPrice = (rp * float64(computeMachineType.MemoryMb) / float64(1024)) + (cp * float64(computeMachineType.GuestCpus))

		err = s.computeMachineTypeRepo.Create(computeMachineTypeTable, transaction, computeMachineType)
		if err != nil {
			s.logger.Error("failed to create compute machine type", zap.Error(err))
			continue
		}
		s.logger.Info("created compute machine type", zap.String("name", mt.Name))
	}

	diskTypes, err := s.fetchDiskTypes(ctx)
	if err != nil {
		s.logger.Error("failed to fetch disk types", zap.Error(err))
		return err
	}
	s.logger.Info("fetched disk types", zap.Any("count", len(types)))
	for _, mt := range diskTypes {
		disk := &model.GCPComputeDiskType{}
		disk.PopulateFromObject(mt)

		region := strings.Join([]string{strings.Split(mt.Zone, "-")[0], strings.Split(mt.Zone, "-")[1]}, "-")
		disk.Region = region

		p, ok := storageTypePrices[mt.Name][region]
		if !ok {
			s.logger.Error("failed to get storage price", zap.String("storage_type", mt.Name), zap.String("region", region),
				zap.Any("prices", storageTypePrices))
			continue
		}

		disk.UnitPrice = p

		err = s.computeDiskTypeRepo.Create(computeDiskTable, transaction, disk)
		if err != nil {
			s.logger.Error("failed to create compute storage type", zap.Error(err))
			continue
		}
		s.logger.Info("created compute storage type", zap.String("name", mt.Name))
	}

	err = s.computeMachineTypeRepo.MoveViewTransaction(computeMachineTypeTable)
	if err != nil {
		return err
	}

	err = s.computeMachineTypeRepo.RemoveOldTables(computeMachineTypeTable)
	if err != nil {
		return err
	}

	err = s.computeDiskTypeRepo.MoveViewTransaction(computeDiskTable)
	if err != nil {
		return err
	}

	err = s.computeDiskTypeRepo.RemoveOldTables(computeDiskTable)
	if err != nil {
		return err
	}

	err = s.computeSKURepo.MoveViewTransaction(computeSKUTable)
	if err != nil {
		return err
	}

	err = s.computeSKURepo.RemoveOldTables(computeSKUTable)
	if err != nil {
		return err
	}

	return nil
}

func (s *GcpService) fetchSKUs(ctx context.Context, service string) ([]*cloudbilling.Sku, error) {
	var results []*cloudbilling.Sku

	err := cloudbilling.NewServicesSkusService(s.apiService).List(fmt.Sprintf("services/%s", service)).Pages(ctx, func(l *cloudbilling.ListSkusResponse) error {
		for _, sku := range l.Skus {
			results = append(results, sku)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *GcpService) fetchMachineTypes(ctx context.Context) ([]*compute.MachineType, error) {
	var results []*compute.MachineType

	zones, err := s.compute.Zones.List(s.project).Do()
	if err != nil {
		return nil, err
	}
	for _, zone := range zones.Items {
		err = s.compute.MachineTypes.List(s.project, zone.Name).Pages(ctx, func(l *compute.MachineTypeList) error {
			for _, mt := range l.Items {
				results = append(results, mt)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (s *GcpService) fetchDiskTypes(ctx context.Context) ([]*compute.DiskType, error) {
	var results []*compute.DiskType

	zones, err := s.compute.Zones.List(s.project).Do()
	if err != nil {
		return nil, err
	}
	for _, zone := range zones.Items {
		err = s.compute.DiskTypes.List(s.project, zone.Name).Pages(ctx, func(l *compute.DiskTypeList) error {
			for _, dt := range l.Items {
				results = append(results, dt)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}
