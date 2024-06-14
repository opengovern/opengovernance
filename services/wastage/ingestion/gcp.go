package ingestion

import (
	"context"
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
	computeSKURepo         repo.GCPComputeSKURepo
}

func NewGcpService(ctx context.Context, logger *zap.Logger, dataAgeRepo repo.DataAgeRepo, db *connector.Database, gcpCredentials string) (*GcpService, error) {
	gcpOpts := []option.ClientOption{
		option.WithCredentialsJSON([]byte(gcpCredentials)),
	}
	gcpOpts = append(gcpOpts, option.WithoutAuthentication())
	apiService, err := cloudbilling.NewService(ctx, gcpOpts...)
	if err != nil {
		return nil, err
	}

	compute, err := compute.NewService(ctx, gcpOpts...)
	if err != nil {
		return nil, err
	}

	return &GcpService{
		logger:      logger,
		DataAgeRepo: dataAgeRepo,
		db:          db,
		apiService:  apiService,
		compute:     compute,
	}, nil
}

func (g *GcpService) IngestComputeInstance(ctx context.Context, tableName string) error {
	var transaction *gorm.DB
	machinteTypePrices := make(map[string]float64)
	skus, err := g.fetchSKUs(ctx, services["ComputeEngine"])
	if err != nil {
		return err
	}
	for _, sku := range skus {
		if sku.PricingInfo == nil || len(sku.PricingInfo) == 0 || sku.PricingInfo[len(sku.PricingInfo)-1].PricingExpression == nil {
			continue
		}

		for _, region := range sku.ServiceRegions {
			computeSKU := &model.GCPComputeSKU{}
			computeSKU.PopulateFromObject(sku, region)

			err = g.computeSKURepo.Create(tableName, transaction, computeSKU)
			if err != nil {
				return err
			}
			if sku.Category.ResourceGroup == cpu || sku.Category.ResourceGroup == ram {
				mf := strings.ToLower(strings.Split(sku.Description, " ")[0])
				machinteTypePrices[fmt.Sprintf("%s.%s", mf, sku.Category.ResourceGroup)] = float64(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Units) +
					(float64(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos) / float64(1_000_000_000))
			}
		}
	}

	types, err := g.fetchMachineTypes(ctx)
	for _, mt := range types {
		computeMachineType := &model.GCPComputeMachineType{}
		computeMachineType.PopulateFromObject(mt)

		mf := strings.ToLower(strings.Split(mt.Name, "-")[0])
		rp, ok := machinteTypePrices[fmt.Sprintf("%s.%s", mf, ram)]
		if !ok {
			continue
		}

		cp, ok := machinteTypePrices[fmt.Sprintf("%s.%s", mf, cpu)]
		if !ok {
			continue
		}

		rp = rp * float64(mt.MemoryMb/1_000)
		cp = cp * float64(mt.GuestCpus)

		computeMachineType.UnitPrice = rp + cp

		err = g.computeMachineTypeRepo.Create(tableName, transaction, computeMachineType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GcpService) fetchSKUs(ctx context.Context, service string) ([]*cloudbilling.Sku, error) {
	var results []*cloudbilling.Sku

	err := cloudbilling.NewServicesSkusService(g.apiService).List(fmt.Sprintf("services/%s", service)).Pages(ctx, func(l *cloudbilling.ListSkusResponse) error {
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

func (g *GcpService) fetchMachineTypes(ctx context.Context) ([]*compute.MachineType, error) {
	var results []*compute.MachineType

	zones, err := g.compute.Zones.List(g.project).Do()
	if err != nil {
		return nil, err
	}
	for _, zone := range zones.Items {
		err = g.compute.MachineTypes.List(g.project, zone.Name).Pages(ctx, func(l *compute.MachineTypeList) error {
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
