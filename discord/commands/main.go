package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/eeekcct/lambda-golang-discord-bot/discord/config"
	"github.com/joho/godotenv"
)

func main() {
	action := flag.String("action", "register", "register or delete")
	flag.Parse()

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	conf := config.NewConfig()
	token, err := GetToken(conf)
	if err != nil {
		log.Println("Error getting token")
		return
	}

	url := fmt.Sprintf(conf.COMMANDS_URL, conf.APPLICATION_ID, conf.GUILD_ID)
	client := NewCommandsClient(url, token)

	switch *action {
	case "register":
		client.RegisterGuildCommands()
	case "delete":
		client.DeleteGuildCommands()
	default:
		log.Println("Invalid action")
	}
}
