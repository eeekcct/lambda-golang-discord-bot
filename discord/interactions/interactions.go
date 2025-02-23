package interactions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/eeekcct/lambda-golang-discord-bot/commands/bedrock"
)

type InteractionResponse struct {
	Content string `json:"content"`
}

type InteractionClient struct {
	ApplicationId    string
	InteractionToken string
}

func NewInteractionClient(applicationId, interactionToken string) *InteractionClient {
	return &InteractionClient{
		ApplicationId:    applicationId,
		InteractionToken: interactionToken,
	}
}

func (i *InteractionClient) InvokeBedrock(ctx context.Context, prompt string) {
	res, err := bedrock.InvokeTitanTextModel(ctx, prompt)
	if err != nil {
		log.Fatal("failed to invoke Titan text model: ", err)
		return
	}
	i.updateMessage(res)
}

func (i *InteractionClient) updateMessage(message string) {
	url := fmt.Sprintf("https://discord.com/api/v10/webhooks/%s/%s/messages/@original", i.ApplicationId, i.InteractionToken)
	data := InteractionResponse{
		Content: message,
	}
	body, err := json.Marshal(data)
	if err != nil {
		log.Println("Failed to marshal data")
		return
	}
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to update Discord messge: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Successfully updated Discord message")
}
