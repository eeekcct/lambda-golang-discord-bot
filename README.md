# Lambda Golang Discord Bot

A Discord bot implemented in Golang, designed to run on AWS Lambda.

## Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Local Development](#local-development)
- [Deployment](#deployment)

## Introduction

This project is a Discord bot written in Golang, designed to be deployed on AWS Lambda. It leverages the Discord API to interact with users and perform various tasks. Currently, it supports only the `/ping` command.

## Features

- **/ping Command**: Responds with "Pong!" when the `/ping` command is issued.
- **Environment Configuration**: Load configuration from environment variables.
- **AWS Lambda Integration**: Seamlessly deploy and run on AWS Lambda.
- **Local Development**: Run and test the bot locally using Docker.

## Prerequisites

Before you begin, ensure you have met the following requirements:

- Go 1.23 or later
- Docker
- AWS CLI
- ngrok (for local development)
- A Discord application with a bot token

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/lambda-golang-discord-bot.git
   cd lambda-golang-discord-bot
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Create a `.env` file in the root directory and add your environment variables:

   ```env
   CLIENT_ID=your_client_id
   CLIENT_SECRET=your_client_secret
   APPLICATION_ID=your_application_id
   GUILD_ID=your_guild_id
   TOKEN_URL=https://discord.com/api/oauth2/token
   COMMANDS_URL=https://discord.com/api/v8/applications/%s/guilds/%s/commands
   DISCORD_PUBLIC_KEY=your_discord_public_key
   ```

## Usage

To register the `/ping` command with Discord:

```bash
go run ./discord/commands
```

To run the bot locally:

1. Start ngrok:

   ```bash
   ngrok http http://localhost:9000
   ```

2. Run the server:

   ```bash
   go run ./cmd/server
   ```

## Local Development

To build and run the bot locally using Docker:

1. Build the Docker image:

   ```bash
   docker build --target local -t lambda-go:local .
   ```

2. Run the Docker container:

   ```bash
   docker run -p 9000:8080 --env-file .env --rm -it lambda-go:local
   ```

3. Test the local deployment:

   ```bash
   curl -X POST "http://localhost:9000/2015-03-31/functions/function/invocations" -H "Content-Type: application/json" -d '{"key": "value"}'
   ```

   **Note**: Ensure that the request is in the format expected by API Gateway. Otherwise, it may result in an error.

## Deployment

To deploy the bot to AWS Lambda, simply run the following Task command:

```bash
task deploy
```
