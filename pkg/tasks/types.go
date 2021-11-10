package tasks

type CloudType string

const (
	CloudTypeAWS   CloudType = "AWS"
	CloudTypeAzure CloudType = "Azure"
)

type AWSAccount struct {
	AccountId   string
	Regions     []string
	Credentials struct {
		SecretKey     string
		AccessKey     string
		SessionToken  string
		AssumeRoleARN string
	}
}

type AzureAccount struct {
	Subscriptions []string
	Credentials   struct {
		TenantID        string
		ClientID        string
		ClientSecret    string
		CertificatePath string
		CertificatePass string
		Username        string
		Password        string
	}
}

type Message struct {
	ResourceType string
	AWS          *AWSAccount
	Azure        *AzureAccount
}

type Config struct {
	Entries []Entry
}

type Entry struct {
	Type  CloudType
	AWS   *AWSAccount
	Azure *AzureAccount
}
