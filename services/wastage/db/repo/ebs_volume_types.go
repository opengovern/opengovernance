package repo

import (
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type EBSVolumeTypeRepo interface {
	Create(tx *gorm.DB, m *model.EBSVolumeType) error
	Get(id uint) (*model.EBSVolumeType, error)
	Update(id uint, m model.EBSVolumeType) error
	Delete(id uint) error
	List() ([]model.EBSVolumeType, error)
	Truncate(tx *gorm.DB) error
	GetCheapestTypeWithSpecs(region string, volumeSize int32, iops int32, throughput float64, validTypes []types.VolumeType) (types.VolumeType, int32, float64, error)
}

type EBSVolumeTypeRepoImpl struct {
	db *connector.Database
}

func NewEBSVolumeTypeRepo(db *connector.Database) EBSVolumeTypeRepo {
	return &EBSVolumeTypeRepoImpl{
		db: db,
	}
}

func (r *EBSVolumeTypeRepoImpl) Create(tx *gorm.DB, m *model.EBSVolumeType) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	return tx.Create(&m).Error
}

func (r *EBSVolumeTypeRepoImpl) Get(id uint) (*model.EBSVolumeType, error) {
	var m model.EBSVolumeType
	tx := r.db.Conn().Model(&model.EBSVolumeType{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *EBSVolumeTypeRepoImpl) Update(id uint, m model.EBSVolumeType) error {
	return r.db.Conn().Model(&model.EBSVolumeType{}).Where("id=?", id).Updates(&m).Error
}

func (r *EBSVolumeTypeRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.EBSVolumeType{}, id).Error
}

func (r *EBSVolumeTypeRepoImpl) List() ([]model.EBSVolumeType, error) {
	var ms []model.EBSVolumeType
	tx := r.db.Conn().Model(&model.EBSVolumeType{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *EBSVolumeTypeRepoImpl) Truncate(tx *gorm.DB) error {
	if tx != nil {
		tx = r.db.Conn()
	}
	tx = tx.Unscoped().Where("1 = 1").Delete(&model.EBSVolumeType{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *EBSVolumeTypeRepoImpl) getDimensionCostsByRegionVolumeTypeAndChargeType(regionCode string, volumeType types.VolumeType, chargeType model.EBSVolumeChargeType) ([]model.EBSVolumeType, error) {
	var m []model.EBSVolumeType
	tx := r.db.Conn().Model(&model.EBSVolumeType{}).
		Where("region_code = ?", regionCode).
		Where("volume_type = ?", volumeType).
		Where("charge_type = ?", chargeType).
		Find(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return m, nil
}

func (r *EBSVolumeTypeRepoImpl) getIo1TotalPrice(region string, volumeSize int32, iops int32) (float64, error) {
	io1IopsPrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeIo1, model.ChargeTypeIOPS)
	if err != nil {
		return 0, err
	}
	io1Iops := 0.0
	for _, iops := range io1IopsPrices {
		io1Iops = iops.PricePerUnit
		break
	}
	io1SizePrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeIo1, model.ChargeTypeSize)
	if err != nil {
		return 0, err
	}
	io1Size := 0.0
	for _, sizes := range io1SizePrices {
		io1Size = sizes.PricePerUnit
		break
	}
	io1Price := io1Iops*float64(iops) + io1Size*float64(volumeSize)

	return io1Price, nil
}

func (r *EBSVolumeTypeRepoImpl) getIo2TotalPrice(region string, volumeSize int32, iops int32) (float64, error) {
	io2IopsPrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeIo2, model.ChargeTypeIOPS)
	if err != nil {
		return 0, err
	}
	io2IopsTier1 := 0.0
	io2IopsTier2 := 0.0
	io2IopsTier3 := 0.0
	for _, iops := range io2IopsPrices {
		switch iops.PriceGroup {
		case "EBS IOPS":
			io2IopsTier1 = iops.PricePerUnit
		case "EBS IOPS Tier 2":
			io2IopsTier2 = iops.PricePerUnit
		case "EBS IOPS Tier 3":
			io2IopsTier3 = iops.PricePerUnit
		}
	}
	io2SizePrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeIo2, model.ChargeTypeSize)
	if err != nil {
		return 0, err
	}
	io2Size := 0.0
	for _, sizes := range io2SizePrices {
		io2Size = sizes.PricePerUnit
		break
	}
	io2Price := io2Size * float64(volumeSize)
	if iops >= model.Io2ProvisionedIopsTier2UpperBound {
		io2Price += io2IopsTier3 * float64(iops-model.Io2ProvisionedIopsTier2UpperBound)
		iops = model.Io2ProvisionedIopsTier2UpperBound
	}
	if iops >= model.Io2ProvisionedIopsTier1UpperBound {
		io2Price += io2IopsTier2 * float64(iops-model.Io2ProvisionedIopsTier1UpperBound)
		iops = model.Io2ProvisionedIopsTier1UpperBound
	}
	io2Price += io2IopsTier1 * float64(iops)

	return io2Price, nil
}

func (r *EBSVolumeTypeRepoImpl) getGp2TotalPrice(region string, volumeSize int32) (float64, error) {
	gp2Prices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeGp2, model.ChargeTypeSize)
	if err != nil {
		return 0, err
	}
	gp2Price := 0.0
	for _, gp2 := range gp2Prices {
		gp2Price = gp2.PricePerUnit
		break
	}
	return gp2Price * float64(volumeSize), nil
}

func (r *EBSVolumeTypeRepoImpl) getGp3TotalPrice(region string, volumeSize int32, iops int32, throughput float64) (float64, error) {
	iops = max(iops-model.Gp3BaseIops, 0)
	throughput = max(throughput-model.Gp3BaseThroughput, 0.0)

	gp3SizePrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeGp3, model.ChargeTypeSize)
	if err != nil {
		return 0, err
	}
	gp3SizePrice := 0.0
	for _, gp3 := range gp3SizePrices {
		gp3SizePrice = gp3.PricePerUnit
		break
	}
	gp3IopsPrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeGp3, model.ChargeTypeIOPS)
	if err != nil {
		return 0, err
	}
	gp3IopsPrice := 0.0
	for _, gp3 := range gp3IopsPrices {
		gp3IopsPrice = gp3.PricePerUnit
		break
	}

	gp3ThroughputPrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeGp3, model.ChargeTypeThroughput)
	if err != nil {
		return 0, err
	}
	gp3ThroughputPrice := 0.0
	for _, gp3 := range gp3ThroughputPrices {
		gp3ThroughputPrice = gp3.PricePerUnit
		break
	}

	return gp3SizePrice*float64(volumeSize) + gp3IopsPrice*float64(iops) + gp3ThroughputPrice*throughput, nil
}

func (r *EBSVolumeTypeRepoImpl) getSc1TotalPrice(region string, volumeSize int32) (float64, error) {
	sc1SizePrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeSc1, model.ChargeTypeSize)
	if err != nil {
		return 0, err
	}
	sc1SizePrice := 0.0
	for _, sc1 := range sc1SizePrices {
		sc1SizePrice = sc1.PricePerUnit
		break
	}

	return sc1SizePrice * float64(volumeSize), nil
}

func (r *EBSVolumeTypeRepoImpl) getSt1TotalPrice(region string, volumeSize int32) (float64, error) {
	st1SizePrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeSt1, model.ChargeTypeSize)
	if err != nil {
		return 0, err
	}
	st1SizePrice := 0.0
	for _, st1 := range st1SizePrices {
		st1SizePrice = st1.PricePerUnit
		break
	}

	return st1SizePrice * float64(volumeSize), nil
}

func (r *EBSVolumeTypeRepoImpl) getStandardTotalPrice(region string, volumeSize int32) (float64, error) {
	standardSizePrices, err := r.getDimensionCostsByRegionVolumeTypeAndChargeType(region, types.VolumeTypeStandard, model.ChargeTypeSize)
	if err != nil {
		return 0, err
	}
	standardSizePrice := 0.0
	for _, standard := range standardSizePrices {
		standardSizePrice = standard.PricePerUnit
		break
	}

	return standardSizePrice * float64(volumeSize), nil
}

func (r *EBSVolumeTypeRepoImpl) getFeasibleVolumeTypes(region string, volumeSize int32, iops int32, throughput float64, validTypes []types.VolumeType) ([]model.EBSVolumeType, error) {
	var res []model.EBSVolumeType
	tx := r.db.Conn().Model(&model.EBSVolumeType{}).Where("region_code = ?", region).
		Where("max_iops >= ?", iops).
		Where("max_throughput >= ?", throughput).
		Where("max_size >= ?", volumeSize)
	if len(validTypes) > 0 {
		tx = tx.Where("volume_type IN ?", validTypes)
	}
	tx = tx.Find(&res)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return res, nil
}

func (r *EBSVolumeTypeRepoImpl) GetCheapestTypeWithSpecs(region string, volumeSize int32, iops int32, throughput float64, validTypes []types.VolumeType) (types.VolumeType, int32, float64, error) {
	volumeTypes, err := r.getFeasibleVolumeTypes(region, volumeSize, iops, throughput, validTypes)
	if err != nil {
		return "", 0, 0, err
	}

	if len(volumeTypes) == 0 {
		return "", 0, 0, errors.New("no feasible volume types found")
	}

	minPrice := 0.0
	resVolumeType := ""
	resBaselineIOPS := int32(0)
	resBaselineThroughput := 0.0
	for _, vt := range volumeTypes {
		var price float64
		var volIops int32
		var volThroughput float64
		switch vt.VolumeType {
		case types.VolumeTypeIo1:
			price, err = r.getIo1TotalPrice(region, volumeSize, iops)
			volIops = 0
			volThroughput = float64(vt.MaxThroughput)
		case types.VolumeTypeIo2:
			price, err = r.getIo2TotalPrice(region, volumeSize, iops)
			volIops = 0
			volThroughput = float64(vt.MaxThroughput)
		case types.VolumeTypeGp2:
			price, err = r.getGp2TotalPrice(region, volumeSize)
			volIops = vt.MaxIops
			volThroughput = float64(vt.MaxThroughput)
		case types.VolumeTypeGp3:
			price, err = r.getGp3TotalPrice(region, volumeSize, iops, throughput)
			volIops = model.Gp3BaseIops
			volThroughput = model.Gp3BaseThroughput
		case types.VolumeTypeSc1:
			price, err = r.getSc1TotalPrice(region, volumeSize)
			volIops = vt.MaxIops
			volThroughput = float64(vt.MaxThroughput)
		case types.VolumeTypeSt1:
			price, err = r.getSt1TotalPrice(region, volumeSize)
			volIops = vt.MaxIops
			volThroughput = float64(vt.MaxThroughput)
		case types.VolumeTypeStandard:
			price, err = r.getStandardTotalPrice(region, volumeSize)
			volIops = vt.MaxIops
			volThroughput = float64(vt.MaxThroughput)
		}
		if err != nil {
			return "", 0, 0, err
		}
		if resVolumeType == "" || price < minPrice {
			minPrice = price
			resVolumeType = string(vt.VolumeType)
			resBaselineIOPS = volIops
			resBaselineThroughput = volThroughput
		}
	}

	return types.VolumeType(resVolumeType), resBaselineIOPS, resBaselineThroughput, nil
}
