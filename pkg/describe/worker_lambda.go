package describe

type LambdaDescribeWorkerInput struct {
	WorkspaceId      string      `json:"workspaceId"`
	WorkspaceName    string      `json:"workspaceName"`
	DescribeEndpoint string      `json:"describeEndpoint"`
	KeyARN           string      `json:"keyARN"`
	KeyRegion        string      `json:"keyRegion"`
	KafkaTopic       string      `json:"kafkaTopic"`
	DescribeJob      DescribeJob `json:"describeJob"`
}
