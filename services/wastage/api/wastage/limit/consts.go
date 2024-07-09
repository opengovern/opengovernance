package limit

const (
	UserEC2InstanceLimit = int32(500)
	UserEBSVolumeLimit   = int32(500)
	UserRDSInstanceLimit = int32(100)
	UserRDSClusterLimit  = int32(50)
	UserAccountLimit     = int32(5)

	OrgEC2InstanceLimit = int32(2000)
	OrgEBSVolumeLimit   = int32(2000)
	OrgRDSInstanceLimit = int32(1000)
	OrgRDSClusterLimit  = int32(500)
	OrgAccountLimit     = int32(5)
)
