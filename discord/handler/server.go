package handler

import (
	"context"

	"github.com/eeekcct/lambda-golang-discord-bot/discord/interactions"
)

func (h *Handler) HandleServer(ctx context.Context, cmd string, prompt string) {
	switch cmd {
	case "bedrock":
		go func() {
			i := interactions.NewInteractionClient(
				h.Config.APPLICATION_ID,
				h.Config.INTERACTION_TOKEN,
			)
			i.InvokeBedrock(ctx, prompt)
		}()
	default:
		return
	}
}
