package describe

type LambdaDescribeWorkerInput struct {
	WorkspaceId      string      `json:"workspaceId"`
	DescribeEndpoint string      `json:"describeEndpoint"`
	KeyARN           string      `json:"keyARN"`
	DescribeJob      DescribeJob `json:"describeJob"`
}
