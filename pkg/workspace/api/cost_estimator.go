package api

import aws "github.com/kaytu-io/kaytu-aws-describer/aws/model"

type GetEC2InstanceCostRequest struct {
	RegionCode string
	Instance   aws.EC2InstanceDescription
}

type GetEC2VolumeCostRequest struct {
	RegionCode string
	Volume     aws.EC2VolumeDescription
}

type GetLBCostRequest struct {
	RegionCode string
	LBType     string
}

type GetRDSInstanceRequest struct {
	RegionCode string
	DBInstance aws.RDSDBInstanceDescription
}
