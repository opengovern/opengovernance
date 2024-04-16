package entity

type AWSCredential struct {
	AccountID string `json:"accountID"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type EC2InstanceWastageRequest struct {
	Credential AWSCredential `json:"credential"`

	InstanceId string `json:"instanceId"`
	Region     string `json:"region"`
}

type EC2InstanceWastageResponse struct {
	CurrentCost float64 `json:"currentCost"`
}
