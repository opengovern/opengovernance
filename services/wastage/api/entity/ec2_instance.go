package entity

type AWSCredential struct {
	AccountID string `json:"accountID"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type Recommendation struct {
	Description string  `json:"description"`
	Saving      float64 `json:"saving"`
}

type EC2InstanceWastageRequest struct {
	Credential AWSCredential `json:"credential"`

	InstanceId string `json:"instanceId"`
	Region     string `json:"region"`
}

type EC2InstanceWastageResponse struct {
	CurrentCost     float64          `json:"currentCost"`
	TotalSavings    float64          `json:"totalSavings"`
	Recommendations []Recommendation `json:"recommendations"`
}
