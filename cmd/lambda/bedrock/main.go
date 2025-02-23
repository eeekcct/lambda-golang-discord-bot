package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/eeekcct/lambda-golang-discord-bot/discord/config"
	"github.com/eeekcct/lambda-golang-discord-bot/discord/interactions"
)

type Request struct {
	Prompt           string `json:"prompt"`
	InteractionToken string `json:"interactionToken"`
}

func Handler(ctx context.Context, event json.RawMessage) error {
	var req Request
	if err := json.Unmarshal(event, &req); err != nil {
		log.Fatal("failed to unmarshal request: ", err)
		return err
	}
	conf := config.NewConfig()
	client := interactions.NewInteractionClient(conf.APPLICATION_ID, req.InteractionToken)
	client.InvokeBedrock(ctx, req.Prompt)
	return nil
}

func main() {
	log.Println("Lambda started")
	lambda.Start(Handler)
}
