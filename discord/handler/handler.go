package handler

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type Interaction struct {
	Type int `json:"type"`
	Data struct {
		Name string `json:"name"`
	} `json:"data"`
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

func HandleRequest(ctx context.Context, body string, headers map[string]string) (int, string) {
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
	if interaction.Data.Name == "ping" {
		log.Println("pong!")
		res := NewInteractionResponse("pong!")
		bytes, err := json.Marshal(res)
		if err != nil {
			log.Println("レスポンスのパースエラー")
			return http.StatusInternalServerError, "Error"
		}
		log.Printf("レスポンス: %s", string(bytes))

		return http.StatusOK, string(bytes)
	}

	return http.StatusInternalServerError, "Error"
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
