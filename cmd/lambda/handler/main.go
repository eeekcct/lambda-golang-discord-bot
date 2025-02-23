package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/eeekcct/lambda-golang-discord-bot/discord/config"
	"github.com/eeekcct/lambda-golang-discord-bot/discord/handler"
)

func HandleLambdaRequest(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	conf := config.NewConfig()
	h := handler.NewHandler(conf)
	status, body := h.HandleRequest(ctx, req.Body, req.Headers, h.HandleLambda)

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
