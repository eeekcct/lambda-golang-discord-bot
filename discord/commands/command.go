package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Command struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Options     []Option `json:"options"`
	Id          string   `json:"id"`
}

type Option struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        int    `json:"type"`
	Required    bool   `json:"required"`
}

type Commands []Command

var commands = Commands{
	{
		Name:        "ping",
		Description: "Replies with pong",
	},
	{
		Name:        "bedrock",
		Description: "Reply from AWS Bedrock",
		Options: []Option{
			{
				Name:        "prompt",
				Description: "Prompt for the Titan text model",
				Type:        3,
				Required:    true,
			},
		},
	},
}

type CommandsClient struct {
	URL   string
	Token string
}

func NewCommandsClient(url, token string) *CommandsClient {
	return &CommandsClient{
		URL:   url,
		Token: token,
	}
}

func (c *CommandsClient) RegisterGuildCommands() {
	for _, command := range commands {
		if err := c.registerGuildCommand(command); err != nil {
			log.Fatalf("Failed to register command %s: %v", command.Name, err)
			return
		}
	}
	log.Println("Successfully registered all commands")
}

func (c *CommandsClient) registerGuildCommand(command Command) error {
	url, token := c.URL, c.Token
	jsonData, err := json.Marshal(command)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Printf("Command %s is already registered\n", command.Name)
	case http.StatusCreated:
		fmt.Printf("Successfully registered command %s\n", command.Name)
	default:
		body, _ := io.ReadAll(resp.Body)
		var res map[string]interface{}
		if err := json.Unmarshal(body, &res); err != nil {
			return fmt.Errorf("failed to register command: unable to parse response")
		}
		return fmt.Errorf("failed to register command: %v", res)
	}
	return nil
}

func (c *CommandsClient) DeleteGuildCommands() {
	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		log.Fatalf("Failed to get commands: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to get commands: %v", err)
		return
	}
	defer resp.Body.Close()

	var commands []Command
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		log.Fatalf("Failed to decode commands: %v", err)
		return
	}

	for _, command := range commands {
		if err := c.deleteGuildCommand(command); err != nil {
			log.Fatalf("Failed to delete command %s: %v", command.Name, err)
			return
		}
	}

	log.Println("Successfully deleted all commands")
}

func (c *CommandsClient) deleteGuildCommand(command Command) error {
	url, token := c.URL+"/"+command.Id, c.Token
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Printf("Successfully deleted command %s\n", command.Name)
	default:
		body, _ := io.ReadAll(resp.Body)
		var res map[string]interface{}
		if err := json.Unmarshal(body, &res); err != nil {
			return fmt.Errorf("failed to delete command: unable to parse response")
		}
		return fmt.Errorf("failed to delete command: %v", res)
	}
	return nil
}
