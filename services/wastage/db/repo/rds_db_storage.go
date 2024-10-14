package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/opengovern/opengovernance/services/wastage/api/entity"
	"github.com/opengovern/opengovernance/services/wastage/db/connector"
	"github.com/opengovern/opengovernance/services/wastage/db/model"
	"github.com/sony/sonyflake"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math"
	"strings"
	"time"
)

type RDSDBStorageRepo interface {
	Create(tableName string, tx *gorm.DB, m *model.RDSDBStorage) error
	Get(id uint) (*model.RDSDBStorage, error)
	Update(id uint, m model.RDSDBStorage) error
	Delete(id uint) error
	List() ([]model.RDSDBStorage, error)
	Truncate(tx *gorm.DB) error
	GetCheapestBySpecs(ctx context.Context, region, engine, edition string, clusterType entity.AwsRdsClusterType, volumeSize, iops int32, throughput float64, validTypes []model.RDSDBStorageVolumeType) (*model.RDSDBStorage, int32, int32, float64, string, error)
	MoveViewTransaction(tableName string) error
	RemoveOldTables(tableName string) error
	CreateNewTable() (string, error)
}

type RDSDBStorageRepoImpl struct {
	logger *zap.Logger
	db     *connector.Database

	viewName string
}

func NewRDSDBStorageRepo(logger *zap.Logger, db *connector.Database) RDSDBStorageRepo {
	stmt := &gorm.Statement{DB: db.Conn()}
	stmt.Parse(&model.RDSDBStorage{})

	return &RDSDBStorageRepoImpl{
		logger: logger,
		db:     db,

		viewName: stmt.Schema.Table,
	}
}

func (r *RDSDBStorageRepoImpl) Create(tableName string, tx *gorm.DB, m *model.RDSDBStorage) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Table(tableName)
	return tx.Create(&m).Error
}

func (r *RDSDBStorageRepoImpl) Get(id uint) (*model.RDSDBStorage, error) {
	var m model.RDSDBStorage
	tx := r.db.Conn().Table(r.viewName).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *RDSDBStorageRepoImpl) Update(id uint, m model.RDSDBStorage) error {
	return r.db.Conn().Model(&model.RDSDBStorage{}).Where("id=?", id).Updates(&m).Error
}

func (r *RDSDBStorageRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.RDSDBStorage{}, id).Error
}

func (r *RDSDBStorageRepoImpl) List() ([]model.RDSDBStorage, error) {
	var ms []model.RDSDBStorage
	tx := r.db.Conn().Table(r.viewName).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *RDSDBStorageRepoImpl) Truncate(tx *gorm.DB) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Unscoped().Where("1 = 1").Delete(&model.RDSDBStorage{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *RDSDBStorageRepoImpl) getMagneticTotalPrice(ctx context.Context, dbStorage model.RDSDBStorage, volumeSize *int32, iops *int32) (float64, string, error) {
	if dbStorage.VolumeType != string(model.RDSDBStorageVolumeTypeMagnetic) {
		return 0, "", errors.New("invalid volume type")
	}
	if dbStorage.MinVolumeSizeGb != 0 && *volumeSize < dbStorage.MinVolumeSizeGb {
		*volumeSize = dbStorage.MinVolumeSizeGb
	}
	sizeCost := dbStorage.PricePerUnit * float64(*volumeSize)

	millionIoPerMonth := math.Ceil(float64(*iops) * 30 * 24 * 60 * 60 / 1e6) // 30 days, 24 hours, 60 minutes, 60 seconds
	iopsCost := 0.0

	tx := r.db.Conn().Table(r.viewName).WithContext(ctx).
		Where("product_family = ?", "System Operation").
		Where("region_code = ?", dbStorage.RegionCode).
		Where("volume_type = ?", "Magnetic").
		Where("'group' = ?", "RDS I/O Operation").
		Where("database_engine IN ?", []string{dbStorage.DatabaseEngine, "Any"})
	tx = tx.Order("price_per_unit asc")
	var iopsStorage model.RDSDBStorage
	err := tx.First(&iopsStorage).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", tx.Error
	}
	iopsCost = iopsStorage.PricePerUnit * millionIoPerMonth

	costBreakdown := fmt.Sprintf("Size: $%.2f * %d\n + IOPS: $%.5f * %.2f (million IOPS per month)", dbStorage.PricePerUnit, *volumeSize, iopsStorage.PricePerUnit, millionIoPerMonth)

	return sizeCost + iopsCost, costBreakdown, nil
}

func (r *RDSDBStorageRepoImpl) getGp2TotalPrice(ctx context.Context, dbStorage model.RDSDBStorage, volumeSize *int32, iops *int32) (float64, string, error) {
	if dbStorage.VolumeType != string(model.RDSDBStorageVolumeTypeGP2) {
		return 0, "", errors.New("invalid volume type")
	}
	if dbStorage.MinVolumeSizeGb != 0 && *volumeSize < dbStorage.MinVolumeSizeGb {
		*volumeSize = dbStorage.MinVolumeSizeGb
	}

	if *iops > 100 {
		minReqSize := int32(math.Ceil(float64(*iops) / model.Gp2IopsPerGiB))
		*volumeSize = max(*volumeSize, minReqSize)
	}
	costBreakdown := fmt.Sprintf("Size: $%.2f * %d", dbStorage.PricePerUnit, *volumeSize)

	return dbStorage.PricePerUnit * float64(*volumeSize), costBreakdown, nil
}

func (r *RDSDBStorageRepoImpl) getGp3TotalPrice(ctx context.Context, clusterType entity.AwsRdsClusterType, dbStorage model.RDSDBStorage, volumeSize *int32, iops *int32, throughput *float64) (float64, string, error) {
	if dbStorage.VolumeType != string(model.RDSDBStorageVolumeTypeGP3) {
		return 0, "", errors.New("invalid volume type")
	}

	getIopsStorage := func(provisionedIops int) (*model.RDSDBStorage, error) {
		tx := r.db.Conn().Table(r.viewName).WithContext(ctx).
			Where("product_family = ?", "Provisioned IOPS").
			Where("region_code = ?", dbStorage.RegionCode).
			Where("deployment_option = ?", dbStorage.DeploymentOption).
			Where("group_description = ?", "RDS Provisioned GP3 IOPS").
			Where("database_engine = ?", dbStorage.DatabaseEngine)
		if len(dbStorage.DatabaseEdition) > 0 {
			tx = tx.Where("database_edition = ?", dbStorage.DatabaseEdition)
		}
		tx = tx.Order("price_per_unit asc")
		var iopsStorage model.RDSDBStorage
		err := tx.First(&iopsStorage).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tx.Error
		}
		return &iopsStorage, nil
	}

	getThroughputStorage := func(provisionedThroughput float64) (*model.RDSDBStorage, error) {
		tx := r.db.Conn().Table(r.viewName).WithContext(ctx).
			Where("product_family = ?", "Provisioned Throughput").
			Where("region_code = ?", dbStorage.RegionCode).
			Where("deployment_option = ?", dbStorage.DeploymentOption).
			Where("database_engine = ?", dbStorage.DatabaseEngine)
		if len(dbStorage.DatabaseEdition) > 0 {
			tx = tx.Where("database_edition = ?", dbStorage.DatabaseEdition)
		}
		tx = tx.Order("price_per_unit asc")
		var throughputStorage model.RDSDBStorage
		err := tx.First(&throughputStorage).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tx.Error
		}
		return &throughputStorage, nil
	}

	dbEngine := strings.ToLower(dbStorage.DatabaseEngine)
	switch {
	case strings.Contains(dbEngine, "sql server"):
		sizeCost := dbStorage.PricePerUnit * float64(*volumeSize)
		costBreakdown := fmt.Sprintf("Size: $%.2f * %d", dbStorage.PricePerUnit, *volumeSize)
		iopsCost := 0.0
		throughputCost := 0.0
		provisionedIops := int(*iops) - model.RDSDBStorageTier1Gp3BaseIops
		provisionedIops = max(provisionedIops, 0)
		if provisionedIops > 0 {
			iopsStorage, err := getIopsStorage(provisionedIops)
			if err != nil {
				return 0, "", err
			}
			iopsCost = iopsStorage.PricePerUnit * float64(provisionedIops)
			costBreakdown += fmt.Sprintf("\n + Provisioned IOPS (anything over %d for sql servers): $%.2f * %d", model.RDSDBStorageTier1Gp3BaseIops, iopsStorage.PricePerUnit, provisionedIops)
		} else {
			*iops = model.RDSDBStorageTier1Gp3BaseIops
		}

		provisionedThroughput := *throughput - model.RDSDBStorageTier1Gp3BaseThroughput
		provisionedThroughput = max(provisionedThroughput, 0)
		if provisionedThroughput > 0 {
			throughputStorage, err := getThroughputStorage(provisionedThroughput)
			if err != nil {
				return 0, "", err
			}
			throughputCost = throughputStorage.PricePerUnit * provisionedThroughput
			costBreakdown += fmt.Sprintf("\n + Provisioned Throughput (anything over %.2f for sql servers): $%.2f * %.2f", model.RDSDBStorageTier1Gp3BaseThroughput, throughputStorage.PricePerUnit, provisionedThroughput)
		} else {
			*throughput = model.RDSDBStorageTier1Gp3BaseThroughput
		}
		return sizeCost + iopsCost + throughputCost, costBreakdown, nil
	default:
		tierThreshold := int32(model.RDSDBStorageTier1Gp3SizeThreshold)
		if strings.Contains(dbEngine, "oracle") {
			tierThreshold = model.RDSDBStorageTier1Gp3SizeThresholdForOracleEngine
		}
		var costBreakdown string
		if *iops > model.RDSDBStorageTier1Gp3BaseIops || *throughput > model.RDSDBStorageTier1Gp3BaseThroughput {
			costBreakdown = fmt.Sprintf("Scaling size to %d to meet IOPS or Throughput requirements", tierThreshold)
			*volumeSize = max(*volumeSize, tierThreshold)
		} else {
			*iops = model.RDSDBStorageTier1Gp3BaseIops
			*throughput = model.RDSDBStorageTier1Gp3BaseThroughput
		}
		if dbStorage.MinVolumeSizeGb != 0 && *volumeSize < dbStorage.MinVolumeSizeGb {
			*volumeSize = dbStorage.MinVolumeSizeGb
		}
		sizeCost := dbStorage.PricePerUnit * float64(*volumeSize)
		iopsCost := 0.0
		throughputCost := 0.0

		if *volumeSize > tierThreshold {
			provisionedIops := int(*iops) - model.RDSDBStorageTier2Gp3BaseIops
			provisionedIops = max(provisionedIops, 0)
			if provisionedIops > 0 {
				iopsStorage, err := getIopsStorage(provisionedIops)
				if err != nil {
					return 0, "", err
				}
				iopsCost = iopsStorage.PricePerUnit * float64(provisionedIops)
				costBreakdown += fmt.Sprintf("\n + Provisioned IOPS (over %d): $%.2f * %d", model.RDSDBStorageTier2Gp3BaseIops, iopsStorage.PricePerUnit, provisionedIops)
			} else {
				*iops = model.RDSDBStorageTier2Gp3BaseIops
			}

			provisionedThroughput := *throughput - model.RDSDBStorageTier2Gp3BaseThroughput
			provisionedThroughput = max(provisionedThroughput, 0)
			switch {
			case clusterType == entity.AwsRdsClusterTypeMultiAzTwoInstance && strings.Contains(dbEngine, "postgres"):
				*throughput = model.RDSDBStorageTier2Gp3BaseThroughput
			case clusterType == entity.AwsRdsClusterTypeMultiAzTwoInstance && strings.Contains(dbEngine, "mysql"):
				*throughput = model.RDSDBStorageTier2Gp3BaseThroughput
				if *iops > model.RDSDBStorageIopsThresholdForThroughputScalingForMySqlEngine {
					*throughput += math.Floor(float64(*iops-model.RDSDBStorageIopsThresholdForThroughputScalingForMySqlEngine) / model.RDSDBStorageThroughputScalingOnIopsFactorForMySqlEngine)
				}
			default:
				if provisionedThroughput > 0 {
					throughputStorage, err := getThroughputStorage(provisionedThroughput)
					if err != nil {
						return 0, "", err
					}
					throughputCost = throughputStorage.PricePerUnit * provisionedThroughput
					costBreakdown += fmt.Sprintf("\n + Provisioned Throughput (over %.2f): $%.2f * %.2f", model.RDSDBStorageTier2Gp3BaseThroughput, throughputStorage.PricePerUnit, provisionedThroughput)
				} else {
					*throughput = model.RDSDBStorageTier2Gp3BaseThroughput
				}
			}
		} // Else is not needed since tier 1 iops/throughput is not configurable and is not charged

		return sizeCost + iopsCost + throughputCost, costBreakdown, nil
	}
}

func (r *RDSDBStorageRepoImpl) getIo1TotalPrice(ctx context.Context, dbStorage model.RDSDBStorage, volumeSize *int32, iops *int32) (float64, string, error) {
	if dbStorage.VolumeType != string(model.RDSDBStorageVolumeTypeIO1) {
		return 0, "", errors.New("invalid volume type")
	}
	if dbStorage.MinVolumeSizeGb != 0 && *volumeSize < dbStorage.MinVolumeSizeGb {
		*volumeSize = dbStorage.MinVolumeSizeGb
	}
	sizeCost := dbStorage.PricePerUnit * float64(*volumeSize)
	iopsCost := 0.0
	tx := r.db.Conn().Table(r.viewName).WithContext(ctx).
		Where("product_family = ?", "Provisioned IOPS").
		Where("region_code = ?", dbStorage.RegionCode).
		Where("deployment_option = ?", dbStorage.DeploymentOption).
		Where("group_description = ?", "RDS Provisioned IOPS").
		Where("database_engine = ?", dbStorage.DatabaseEngine)
	if len(dbStorage.DatabaseEdition) > 0 {
		tx = tx.Where("database_edition = ?", dbStorage.DatabaseEdition)
	}
	tx = tx.Order("price_per_unit asc")
	var iopsStorage model.RDSDBStorage
	err := tx.First(&iopsStorage).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", tx.Error
	}
	iopsCost = iopsStorage.PricePerUnit * float64(*iops)

	costBreakdown := fmt.Sprintf("Size: $%.2f * %d\n + IOPS: $%.2f * %d", dbStorage.PricePerUnit, *volumeSize, iopsStorage.PricePerUnit, *iops)

	return sizeCost + iopsCost, costBreakdown, nil
}

func (r *RDSDBStorageRepoImpl) getIo2TotalPrice(ctx context.Context, dbStorage model.RDSDBStorage, volumeSize *int32, iops *int32) (float64, string, error) {
	if dbStorage.VolumeType != string(model.RDSDBStorageVolumeTypeIO2) {
		return 0, "", errors.New("invalid volume type")
	}
	if dbStorage.MinVolumeSizeGb != 0 && *volumeSize < dbStorage.MinVolumeSizeGb {
		*volumeSize = dbStorage.MinVolumeSizeGb
	}
	sizeCost := dbStorage.PricePerUnit * float64(*volumeSize)
	iopsCost := 0.0
	tx := r.db.Conn().Table(r.viewName).WithContext(ctx).
		Where("product_family = ?", "Provisioned IOPS").
		Where("region_code = ?", dbStorage.RegionCode).
		Where("deployment_option = ?", dbStorage.DeploymentOption).
		Where("group_description = ?", "RDS Provisioned IO2 IOPS").
		Where("database_engine = ?", dbStorage.DatabaseEngine)
	if len(dbStorage.DatabaseEdition) > 0 {
		tx = tx.Where("database_edition = ?", dbStorage.DatabaseEdition)
	}
	tx = tx.Order("price_per_unit asc")
	var iopsStorage model.RDSDBStorage
	err := tx.First(&iopsStorage).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", tx.Error
	}
	iopsCost = iopsStorage.PricePerUnit * float64(*iops)

	costBreakdown := fmt.Sprintf("Size: $%.2f * %d\n + IOPS: $%.2f * %d", dbStorage.PricePerUnit, *volumeSize, iopsStorage.PricePerUnit, *iops)

	return sizeCost + iopsCost, costBreakdown, nil
}

func (r *RDSDBStorageRepoImpl) getAuroraGeneralPurposeTotalPrice(ctx context.Context, dbStorage model.RDSDBStorage, volumeSize *int32, iops *int32) (float64, string, error) {
	if dbStorage.VolumeType != string(model.RDSDBStorageVolumeTypeGeneralPurposeAurora) {
		return 0, "", errors.New("invalid volume type")
	}
	// Disable min volume size check for aurora since use is not managing it
	//if dbStorage.MinVolumeSizeGb != 0 && *volumeSize < dbStorage.MinVolumeSizeGb {
	//	*volumeSize = dbStorage.MinVolumeSizeGb
	//}
	sizeCost := dbStorage.PricePerUnit * float64(*volumeSize)

	millionIoPerMonth := math.Ceil(float64(*iops) * 30 * 24 * 60 * 60 / 1e6) // 30 days, 24 hours, 60 minutes, 60 seconds
	iopsCost := 0.0

	tx := r.db.Conn().Table(r.viewName).WithContext(ctx).
		Where("product_family = ?", "System Operation").
		Where("region_code = ?", dbStorage.RegionCode).
		Where("'group' = ?", "Aurora I/O Operation").
		Where("database_engine IN ?", []string{dbStorage.DatabaseEngine, "Any"})
	tx = tx.Order("price_per_unit asc")
	var iopsStorage model.RDSDBStorage
	err := tx.First(&iopsStorage).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", tx.Error
	}
	iopsCost = iopsStorage.PricePerUnit * millionIoPerMonth

	costBreakdown := fmt.Sprintf("Size: $%.2f * %d\n + IOPS: $%.5f * %.2f (million IOPS per month)", dbStorage.PricePerUnit, *volumeSize, iopsStorage.PricePerUnit, millionIoPerMonth)

	return sizeCost + iopsCost, costBreakdown, nil

}

func (r *RDSDBStorageRepoImpl) getAuroraIOOptimizedTotalPrice(ctx context.Context, dbStorage model.RDSDBStorage, volumeSize *int32) (float64, string, error) {
	if dbStorage.VolumeType != string(model.RDSDBStorageVolumeTypeIOOptimizedAurora) {
		return 0, "", errors.New("invalid volume type")
	}
	// Disable min volume size check for aurora since use is not managing it
	//if dbStorage.MinVolumeSizeGb != 0 && *volumeSize < dbStorage.MinVolumeSizeGb {
	//	*volumeSize = dbStorage.MinVolumeSizeGb
	//}
	sizeCost := dbStorage.PricePerUnit * float64(*volumeSize)

	costBreakdown := fmt.Sprintf("Size: $%.2f * %d", dbStorage.PricePerUnit, *volumeSize)

	return sizeCost, costBreakdown, nil
}

func (r *RDSDBStorageRepoImpl) getFeasibleVolumeTypes(ctx context.Context, region string, engine, edition string, clusterType entity.AwsRdsClusterType, volumeSize int32, iops int32, throughput float64, validTypes []model.RDSDBStorageVolumeType) ([]model.RDSDBStorage, error) {
	var res []model.RDSDBStorage
	tx := r.db.Conn().Table(r.viewName).WithContext(ctx).
		Where("product_family = ?", "Database Storage").
		Where("region_code = ?", region).
		Where("max_volume_size_gb >= ? or max_volume_size = ''", volumeSize).
		Where("max_iops >= ?", iops).
		Where("max_throughput_mb >= ?", throughput)

	if strings.Contains(strings.ToLower(engine), "aurora") {
		var filteredValidTypes []model.RDSDBStorageVolumeType
		for _, t := range validTypes {
			if t == model.RDSDBStorageVolumeTypeIOOptimizedAurora ||
				t == model.RDSDBStorageVolumeTypeGeneralPurposeAurora {
				filteredValidTypes = append(filteredValidTypes, t)
			}
		}
		if len(filteredValidTypes) == 0 {
			filteredValidTypes = []model.RDSDBStorageVolumeType{
				model.RDSDBStorageVolumeTypeIOOptimizedAurora,
				model.RDSDBStorageVolumeTypeGeneralPurposeAurora,
			}
		}
		validTypes = filteredValidTypes
		tx = tx.Where("database_engine IN ?", []string{engine, "Any"})
		tx = tx.Where("deployment_option = ?", "Single-AZ")
	} else {
		var filteredValidTypes []model.RDSDBStorageVolumeType
		for _, t := range validTypes {
			if t != model.RDSDBStorageVolumeTypeIOOptimizedAurora &&
				t != model.RDSDBStorageVolumeTypeGeneralPurposeAurora {
				filteredValidTypes = append(filteredValidTypes, t)
			}
		}
		validTypes = filteredValidTypes
		tx = tx.Where("database_engine = ?", engine)
		if len(edition) > 0 {
			tx = tx.Where("database_edition = ?", edition)
		}
		tx = tx.Where("deployment_option = ?", string(clusterType))
	}

	if len(validTypes) > 0 {
		tx = tx.Where("volume_type IN ?", validTypes)
	}

	tx = tx.Find(&res)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return res, nil
}

func (r *RDSDBStorageRepoImpl) GetCheapestBySpecs(ctx context.Context, region, engine, edition string, clusterType entity.AwsRdsClusterType, volumeSize, iops int32, throughput float64, validTypes []model.RDSDBStorageVolumeType) (res *model.RDSDBStorage, cheapestVSize, cheapestIops int32, cheapestThroughput float64, cheapestCostBreakdown string, err error) {
	res = nil
	err = nil
	cheapestVSize = volumeSize
	cheapestIops = iops
	cheapestThroughput = throughput
	cheapestCostBreakdown = ""

	volumes, err := r.getFeasibleVolumeTypes(ctx, region, engine, edition, clusterType, volumeSize, iops, throughput, validTypes)
	if err != nil {
		return nil, 0, 0, 0, "", err
	}

	if len(volumes) == 0 {
		return nil, 0, 0, 0, "", nil
	}

	var cheapestPrice float64

	for _, v := range volumes {
		v := v
		vSize := volumeSize
		vIops := iops
		vThroughput := throughput
		vCostBreakdown := ""
		var totalCost float64
		switch model.RDSDBStorageVolumeType(v.VolumeType) {
		case model.RDSDBStorageVolumeTypeMagnetic:
			totalCost, vCostBreakdown, err = r.getMagneticTotalPrice(ctx, v, &vSize, &vIops)
		case model.RDSDBStorageVolumeTypeGP2:
			totalCost, vCostBreakdown, err = r.getGp2TotalPrice(ctx, v, &vSize, &vIops)
		case model.RDSDBStorageVolumeTypeGP3:
			totalCost, vCostBreakdown, err = r.getGp3TotalPrice(ctx, clusterType, v, &vSize, &vIops, &vThroughput)
		case model.RDSDBStorageVolumeTypeIO1:
			totalCost, vCostBreakdown, err = r.getIo1TotalPrice(ctx, v, &vSize, &vIops)
		case model.RDSDBStorageVolumeTypeIO2:
			totalCost, vCostBreakdown, err = r.getIo2TotalPrice(ctx, v, &vSize, &vIops)
		case model.RDSDBStorageVolumeTypeGeneralPurposeAurora:
			totalCost, vCostBreakdown, err = r.getAuroraGeneralPurposeTotalPrice(ctx, v, &vSize, &vIops)
		case model.RDSDBStorageVolumeTypeIOOptimizedAurora:
			totalCost, vCostBreakdown, err = r.getAuroraIOOptimizedTotalPrice(ctx, v, &vSize)
		}

		if err != nil {
			r.logger.Error("failed to calculate total cost", zap.Error(err), zap.Any("volume", v))
			return nil, 0, 0, 0, "", err
		}

		if res == nil || totalCost < cheapestPrice {
			res = &v
			cheapestVSize = vSize
			cheapestIops = vIops
			cheapestThroughput = vThroughput
			cheapestCostBreakdown = vCostBreakdown
			cheapestPrice = totalCost
		}
	}

	return res, cheapestVSize, cheapestIops, cheapestThroughput, cheapestCostBreakdown, nil
}

func (r *RDSDBStorageRepoImpl) CreateNewTable() (string, error) {
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	var ec2InstanceTypeTable string
	for {
		id, err := sf.NextID()
		if err != nil {
			return "", err
		}

		ec2InstanceTypeTable = fmt.Sprintf("%s_%s_%d",
			r.viewName,
			time.Now().Format("2006_01_02"),
			id,
		)
		var c int32
		tx := r.db.Conn().Raw(fmt.Sprintf(`
		SELECT count(*)
		FROM information_schema.tables
		WHERE table_schema = current_schema
		AND table_name = '%s';
	`, ec2InstanceTypeTable)).First(&c)
		if tx.Error != nil {
			return "", err
		}
		if c == 0 {
			break
		}
	}

	err := r.db.Conn().Table(ec2InstanceTypeTable).AutoMigrate(&model.RDSDBStorage{})
	if err != nil {
		return "", err
	}
	return ec2InstanceTypeTable, nil
}

func (r *RDSDBStorageRepoImpl) MoveViewTransaction(tableName string) error {
	tx := r.db.Conn().Begin()
	var err error
	defer func() {
		_ = tx.Rollback()
	}()

	dropViewQuery := fmt.Sprintf("DROP VIEW IF EXISTS rdsdb_storages")
	tx = tx.Exec(dropViewQuery)
	err = tx.Error
	if err != nil {
		return err
	}

	createViewQuery := fmt.Sprintf(`
  CREATE OR REPLACE VIEW rdsdb_storages AS
  SELECT *
  FROM %s;
`, tableName)

	tx = tx.Exec(createViewQuery)
	err = tx.Error
	if err != nil {
		return err
	}

	tx = tx.Commit()
	err = tx.Error
	if err != nil {
		return err
	}
	return nil
}

func (r *RDSDBStorageRepoImpl) getOldTables(currentTableName string) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = current_schema
		AND table_name LIKE 'rdsdb_storages_%%' AND table_name <> '%s';
	`, currentTableName)

	var tableNames []string
	tx := r.db.Conn().Raw(query).Find(&tableNames)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tableNames, nil
}

func (r *RDSDBStorageRepoImpl) RemoveOldTables(currentTableName string) error {
	tableNames, err := r.getOldTables(currentTableName)
	if err != nil {
		return err
	}
	for _, tn := range tableNames {
		err = r.db.Conn().Migrator().DropTable(tn)
		if err != nil {
			return err
		}
	}
	return nil
}
