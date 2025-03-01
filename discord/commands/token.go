package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eeekcct/lambda-golang-discord-bot/discord/config"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func GetToken(conf *config.Config) (string, error) {
	data := "grant_type=client_credentials&scope=applications.commands.update"
	req, err := http.NewRequest("POST", conf.TOKEN_URL, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(conf.CLIENT_ID, conf.CLIENT_SECRET)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get token: %s", string(body))
	}

	var token TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}
