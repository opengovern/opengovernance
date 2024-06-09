package ingestion

import (
	"context"
	"fmt"
	"github.com/cycloidio/terracost/price"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

var (
	services = map[string]string{
		"ComputeEngine": "6F81-5844-456A",
	}
)

type GcpService struct {
	logger *zap.Logger

	apiService *cloudbilling.APIService
	compute    *compute.Service

	DataAgeRepo repo.DataAgeRepo

	db                     *connector.Database
	computeMachineTypeRepo repo.GCPComputeMachineTypeRepo
	computeSKURepo         repo.GCPComputeSKURepo

	err error
}

func NewGcpService(ctx context.Context, logger *zap.Logger, dataAgeRepo repo.DataAgeRepo, db *connector.Database) (*GcpService, error) {
	gcpOpts := []option.ClientOption{
		option.WithCredentialsJSON([]byte(`{"type": "service_account"}`)),
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

	go func() {
		defer close(results)

		// Key is the "machineFamily.(cpu|ram)" ex (e2.cpu) and value the price of it
		// it is used after the imports of the SKU as we also import the MachineTypes
		// and to calculate the prices we need this
		var (
			machinteTypePrices = make(map[string]price.Price)
		)

		for sku := range g.fetchSKUs(ctx, services["ComputeEngine"]) {
			select {
			case <-ctx.Done():
				g.err = ctx.Err()
				return
			default:
			}

			// If the SKU has no price we do not need it
			// IDK if this is possible but this way we
			// are sure that we'll not ingest it.
			// As it's ordered in chronological order (of sku.PricingInfo) we take
			// the last one as it would be the most current and validate
			// if it has a PricingExpression
			if sku.PricingInfo == nil || len(sku.PricingInfo) == 0 || sku.PricingInfo[len(sku.PricingInfo)-1].PricingExpression == nil {
				continue
			}

			for _, region := range sku.ServiceRegions {
				// Google has a region named 'global' which means the price is applied to all regions
				computeSKU := &model.GCPComputeSKU{}
				computeSKU.PopulateFromObject(sku, region)

				g.computeSKURepo.Create(tableName, transaction, computeSKU)
			}
		}

		for mt := range ing.fetchMachineTypes(ctx) {
			mf := strings.ToLower(strings.Split(mt.Name, "-")[0])
			_, ok := machineFamilies[mf]
			if !ok {
				continue
			}

			prod := &product.Product{
				Provider: ProviderName,
				SKU:      fmt.Sprintf("machine-types-%d", mt.Id),
				Service:  "Compute Engine",
				Family:   "Compute",
				Location: ing.region,
				Attributes: map[string]string{
					"machine_type":   mt.Name,
					"group":          "MachineType",
					"cpu":            strconv.Itoa(int(mt.GuestCpus)),
					"ram":            strconv.Itoa(int(mt.MemoryMb)),
					"kind":           mt.Kind,
					"machine_family": mf,
				},
			}

			rp, ok := machinteTypePrices[fmt.Sprintf("%s.%s", mf, ram)]
			if !ok {
				continue
			}

			cp, ok := machinteTypePrices[fmt.Sprintf("%s.%s", mf, cpu)]
			if !ok {
				continue
			}

			// The mt.MemboryMb is in MB and the rp.Unit is on GiBy
			// so we have to convert it to the same unit
			rp.Value = rp.Value.Mul(decimal.NewFromInt(mt.MemoryMb / 1_000))
			cp.Value = cp.Value.Mul(decimal.NewFromInt(mt.GuestCpus))

			// The Unit for the RAM is in GiBy.h, but for the addition
			// we need the same unit which in this case would be h
			if rp.Unit == "GiBy.h" {
				rp.Unit = "h"
			} else {
				// If we cannot check the unit we skip it
				continue
			}
			err := rp.Add(cp)
			if err != nil {
				ing.err = err
				return
			}

			pwp := &price.WithProduct{
				// rp has the end result of the Sum of the 2 prices
				Price:   rp,
				Product: prod,
			}

			if ing.ingestionFilter(pwp) {
				results <- pwp
			}
		}
	}()

	return results
}

func (g *GcpService) fetchSKUs(ctx context.Context, service string) <-chan *cloudbilling.Sku {
	results := make(chan *cloudbilling.Sku, 100)

	go func() {
		defer close(results)
		err := cloudbilling.NewServicesSkusService(g.apiService).List(fmt.Sprintf("services/%s", service)).Pages(ctx, func(l *cloudbilling.ListSkusResponse) error {
			for _, sku := range l.Skus {
				results <- sku
			}
			return nil
		})
		if err != nil {
			g.err = err
		}
	}()

	return results
}

func (g *GcpService) fetchMachineTypes(ctx context.Context) <-chan *compute.MachineType {
	results := make(chan *compute.MachineType, 100)

	go func() {
		defer close(results)
		err := g.compute.MachineTypes.List(ing.project, ing.zone).Pages(ctx, func(l *compute.MachineTypeList) error {
			for _, mt := range l.Items {
				results <- mt
			}
			return nil
		})
		if err != nil {
			g.err = err
		}
	}()

	return results
}
