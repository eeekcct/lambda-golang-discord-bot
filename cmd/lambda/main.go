package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/eeekcct/lambda-golang-discord-bot/discord/handler"
)

func HandleLambdaRequest(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	status, body := handler.HandleRequest(ctx, req.Body, req.Headers)

	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Body:       body,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

func main() {
	log.Println("Lambda started")
	lambda.Start(HandleLambdaRequest)
}
