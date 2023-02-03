package api

type CloudNativeConnectionWorkerTriggerInput struct {
	JobID                   string `json:"jobId,omitempty"`
	JobJson                 string `json:"jobJson,omitempty"`
	EndOfJobCallbackURL     string `json:"endOfJobCallbackUrl,omitempty"`
	CredentialsCallbackURL  string `json:"credentialsCallbackUrl,omitempty"`
	CredentialDecryptionKey string `json:"credentialDecryptionKey,omitempty"`
	OutputEncryptionKey     string `json:"outputEncryptionKey,omitempty"`
	ResourcesTopic          string `json:"resourcesTopic,omitempty"`
}

type CloudNativeConnectionWorkerTriggerOutput struct {
	ID                    string `json:"id,omitempty"`
	StatusQueryGetURI     string `json:"statusQueryGetUri,omitempty"`
	SendEventPostURI      string `json:"sendEventPostUri,omitempty"`
	TerminatePostURI      string `json:"terminatePostUri,omitempty"`
	RewindPostURI         string `json:"rewindPostUri,omitempty"`
	PurgeHistoryDeleteURI string `json:"purgeHistoryDeleteUri,omitempty"`
	RestartPostURI        string `json:"restartPostUri,omitempty"`
	SuspendPostURI        string `json:"suspendPostUri,omitempty"`
	ResumePostURI         string `json:"resumePostUri,omitempty"`
}
