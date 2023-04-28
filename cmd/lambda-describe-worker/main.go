package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
)

func main() {
	lambda.Start(describe.DescribeHandler)
}
