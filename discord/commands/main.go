package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	config := NewConfig()
	token, err := GetToken(config)
	if err != nil {
		log.Println("Error getting token")
		return
	}

	url := fmt.Sprintf(config.COMMANDS_URL, config.APPLICATION_ID, config.GUILD_ID)
	err = RegisterCommands(url, token)
	if err != nil {
		log.Printf("Error registering commands: %s", err)
		return
	}
	log.Println("Successfully registered commands")
}
