package main

import (
	"expo-open-ota/internal/handlers"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handlers.LambdaAssetsHandler)
}
