package entity

type Configuration struct {
	EC2LazyLoad int `json:"ec2LazyLoad"`
	RDSLazyLoad int `json:"rdsLazyLoad"`
}
