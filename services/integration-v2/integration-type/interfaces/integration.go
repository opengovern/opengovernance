package interfaces

type IntegrationCreator func(certificateType string, jsonData []byte) (CredentialType, map[string]any, error)
