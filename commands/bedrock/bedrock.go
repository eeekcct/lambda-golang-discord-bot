package bedrock

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type InvokeModelWrapper struct {
	Client  *bedrockruntime.Client
	ModelId string
}

type TitanTextRequest struct {
	InputText            string               `json:"inputText"`
	TextGenerationConfig TextGenerationConfig `json:"textGenerationConfig"`
}

type TextGenerationConfig struct {
	Temperature   float64  `json:"temperature"`
	TopP          float64  `json:"topP"`
	MaxTokenCount int      `json:"maxTokenCount"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type TitanTextResponse struct {
	InputTextTokenCount int      `json:"inputTextTokenCount"`
	Results             []Result `json:"results"`
}

type Result struct {
	TokenCount       int    `json:"tokenCount"`
	OutputText       string `json:"outputText"`
	CompletionReason string `json:"completionReason"`
}

func NewInvokeModelWrapper(config aws.Config, id string) *InvokeModelWrapper {
	return &InvokeModelWrapper{
		Client:  bedrockruntime.NewFromConfig(config),
		ModelId: id,
	}
}

func (i *InvokeModelWrapper) InvokeTitanText(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(TitanTextRequest{
		InputText: prompt,
		TextGenerationConfig: TextGenerationConfig{
			Temperature:   0.7, // Default value
			TopP:          0.9, // Default value
			MaxTokenCount: 512, // Default value
		},
	})
	if err != nil {
		log.Fatal("failed to marshal request body: ", err)
		return "", err
	}
	output, err := i.Client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(i.ModelId),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		log.Fatal("failed to invoke model: ", err)
		return "", err
	}

	var response TitanTextResponse
	if err := json.Unmarshal(output.Body, &response); err != nil {
		log.Fatal("failed to unmarshal response: ", err)
		return "", err
	}

	return response.Results[0].OutputText, nil
}

func InvokeTitanTextModel(ctx context.Context, prompt string) (string, error) {
	// Set the AWS Region that the service clients should use
	config, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("ap-northeast-1"))
	if err != nil {
		log.Fatal("failed to load configuration: ", err)
		return "", err
	}

	// Create a new client
	client := NewInvokeModelWrapper(config, "amazon.titan-text-express-v1")

	res, err := client.InvokeTitanText(ctx, prompt)
	if err != nil {
		log.Fatal("failed to invoke model: ", err)
		return "", err
	}
	return res, nil
}
