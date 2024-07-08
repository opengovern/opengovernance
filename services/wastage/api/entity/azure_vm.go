package entity

type AzureVM struct {
	Id           string `json:"id"`
	Zone         string `json:"zone"`
	Region       string `json:"region"`
	InstanceType string `json:"instance_type"`
}
