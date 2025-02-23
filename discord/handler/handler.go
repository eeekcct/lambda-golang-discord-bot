package handler

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/eeekcct/lambda-golang-discord-bot/discord/config"
)

type Interaction struct {
	Type  int    `json:"type"`
	Token string `json:"token"`
	Data  struct {
		Name    string   `json:"name"`
		Options []Option `json:"options"`
	} `json:"data"`
}

type Option struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type InteractionResponse struct {
	Type int `json:"type"`
	Data struct {
		Content string `json:"content"`
	} `json:"data"`
}

func NewInteractionResponse(content string) *InteractionResponse {
	return &InteractionResponse{
		Type: 4,
		Data: struct {
			Content string `json:"content"`
		}{
			Content: content,
		},
	}
}

type Handler struct {
	Config *config.Config
}

func NewHandler(config *config.Config) *Handler {
	return &Handler{
		Config: config,
	}
}

func (h *Handler) HandleRequest(ctx context.Context, body string, headers map[string]string, process func(context.Context, string, string)) (int, string) {
	var err error
	key := os.Getenv("DISCORD_PUBLIC_KEY")
	decodedKey, err := hex.DecodeString(key)
	if err != nil {
		log.Println("公開鍵のデコードエラー")
		return http.StatusInternalServerError, "Error"
	}

	// Discordの署名検証
	if !verifySignature(headers, body, decodedKey) {
		log.Println("署名検証エラー")
		return http.StatusUnauthorized, "Unauthorized"
	}

	// リクエストのBodyをパース
	var interaction Interaction
	err = json.Unmarshal([]byte(body), &interaction)
	if err != nil {
		log.Println("リクエストのパースエラー")
		return http.StatusInternalServerError, "Error"
	}

	// PINGリスクエストへの応答（Discordの使用で必要）
	if interaction.Type == 1 {
		log.Println("PINGリクエスト")
		return http.StatusOK, `{"type": 1}`
	}

	// PINGリスクエスト以外のリクエストへの応答
	h.Config.INTERACTION_TOKEN = interaction.Token
	var res *InteractionResponse
	switch interaction.Data.Name {
	case "ping":
		log.Println("pong!")
		res = NewInteractionResponse("pong!")
	case "bedrock":
		log.Println("AWS Bedrockからのリクエスト")
		prompt := interaction.Data.Options[0].Value
		process(ctx, "bedrock", prompt)
		return http.StatusOK, `{"type": 5}`
	default:
		log.Println("不明なリクエスト")
		return http.StatusInternalServerError, "Error"
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		log.Println("レスポンスのパースエラー")
		return http.StatusInternalServerError, "Error"
	}
	log.Printf("レスポンス: %s", string(bytes))
	return http.StatusOK, string(bytes)
}

// Discordの署名検証をする関数
func verifySignature(headers map[string]string, body string, pubicKey ed25519.PublicKey) bool {
	signature := headers["x-signature-ed25519"]
	timestamp := headers["x-signature-timestamp"]
	if signature == "" || timestamp == "" {
		log.Println("署名がありません")
		return false
	}
	message := []byte(timestamp + body)
	sig, err := hex.DecodeString(signature)
	if err != nil {
		log.Println("署名のデコードエラー")
		return false
	}

	// Discordの署名検証
	return ed25519.Verify(pubicKey, message, sig)
}
