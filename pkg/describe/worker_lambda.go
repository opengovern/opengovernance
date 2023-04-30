package describe

type LambdaDescribeWorkerInput struct {
	WorkspaceId      string      `json:"workspaceId"`
	DescribeEndpoint string      `json:"describeEndpoint"`
	KeyARN           string      `json:"keyARN"`
	KeyRegion        string      `json:"keyRegion"`
	DescribeJob      DescribeJob `json:"describeJob"`
}
