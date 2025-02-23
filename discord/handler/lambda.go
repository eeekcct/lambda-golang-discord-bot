package handler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaType "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func (h *Handler) HandleLambda(ctx context.Context, cmd string, prompt string) {
	switch cmd {
	case "bedrock":
		h.invokeLambda(ctx, "bedrock", prompt)
	default:
		return
	}
}

func (h *Handler) invokeLambda(ctx context.Context, functionName string, prompt string) {
	sdkConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(h.Config.AWS_REGION))
	if err != nil {
		log.Fatal("failed to create session: ", err)
		return
	}
	client := lambda.NewFromConfig(sdkConfig)
	payload, err := json.Marshal(map[string]interface{}{
		"Prompt":           prompt,
		"InteractionToken": h.Config.INTERACTION_TOKEN,
	})
	if err != nil {
		log.Fatal("failed to marshal request body: ", err)
		return
	}
	output, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: lambdaType.InvocationTypeEvent,
		Payload:        payload,
	})
	if err != nil {
		log.Fatal("failed to invoke function: ", err)
		return
	}
	log.Println("Invoke output: ", output)
}
