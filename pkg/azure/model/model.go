//go:generate go run ../../keibi-es-sdk/gen/main.go --file $GOFILE --output ../../keibi-es-sdk/azure_resources_clients.go --type azure

package model

type Metadata struct {
	SubscriptionID   string `json:"subscription_id"`
	CloudEnvironment string `json:"cloud_environment"`
}
