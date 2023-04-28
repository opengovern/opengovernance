package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"go.uber.org/zap"
	"log"
)

func DescribeHandler(ctx context.Context) {
	lc, _ := lambdacontext.FromContext(ctx)
	log.Print(lc.Identity.CognitoIdentityPoolID)

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println(err)
		return
	}

	w, err := describe.InitializeLambdaDescribeWorker(
		workspaceId,
		connectionId,
		resourceType,
		describeEndpoint,
		logger,
	)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {
	lambda.Start(DescribeHandler)
}
