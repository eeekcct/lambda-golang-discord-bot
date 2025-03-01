package config

import (
	"os"
)

type Config struct {
	CLIENT_ID          string
	CLIENT_SECRET      string
	APPLICATION_ID     string
	GUILD_ID           string
	TOKEN_URL          string
	COMMANDS_URL       string
	DISCORD_PUBLIC_KEY string
	INTERACTION_TOKEN  string
	AWS_REGION         string
}

func NewConfig() *Config {
	client_id := os.Getenv("CLIENT_ID")
	client_secret := os.Getenv("CLIENT_SECRET")
	application_id := os.Getenv("APPLICATION_ID")
	guild_id := os.Getenv("GUILD_ID")
	token_url := os.Getenv("TOKEN_URL")
	commands_url := os.Getenv("COMMANDS_URL")
	discord_public_key := os.Getenv("DISCORD_PUBLIC_KEY")
	aws_region := os.Getenv("AWS_REGION")

	return &Config{
		CLIENT_ID:          client_id,
		CLIENT_SECRET:      client_secret,
		APPLICATION_ID:     application_id,
		GUILD_ID:           guild_id,
		TOKEN_URL:          token_url,
		COMMANDS_URL:       commands_url,
		DISCORD_PUBLIC_KEY: discord_public_key,
		AWS_REGION:         aws_region,
	}
}
