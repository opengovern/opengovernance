package wastage

var (
	UserEC2InstanceLimit = int32(500)
	UserEBSVolumeLimit   = int32(500)
	UserRDSInstanceLimit = int32(50) //temporary changed for testing, should be 100
	UserRDSClusterLimit  = int32(50)
	UserAccountLimit     = int32(5)

	OrgEC2InstanceLimit = int32(2000)
	OrgEBSVolumeLimit   = int32(2000)
	OrgRDSInstanceLimit = int32(50) //temporary changed for testing, should be 1000
	OrgRDSClusterLimit  = int32(500)
	OrgAccountLimit     = int32(5)
)
