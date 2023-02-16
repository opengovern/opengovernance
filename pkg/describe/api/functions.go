package api

type CloudNativeConnectionWorkerTriggerInput struct {
	WorkspaceID             string `json:"workspaceId,omitempty"`
	JobID                   string `json:"jobId,omitempty"`
	JobJson                 string `json:"jobJson,omitempty"`
	EndOfJobCallbackURL     string `json:"endOfJobCallbackUrl,omitempty"`
	CredentialsCallbackURL  string `json:"credentialsCallbackUrl,omitempty"`
	CredentialDecryptionKey string `json:"credentialDecryptionKey,omitempty"`
	OutputEncryptionKey     string `json:"outputEncryptionKey,omitempty"`
	ResourcesTopic          string `json:"resourcesTopic,omitempty"`
}

type CloudNativeConnectionWorkerTriggerQueueMessage struct {
	JobId  string `json:"jobId,omitempty"`
	Status int    `json:"status,omitempty"`
	Body   string `json:"body,omitempty"`
}

type CloudNativeConnectionWorkerTriggerOutput struct {
}
