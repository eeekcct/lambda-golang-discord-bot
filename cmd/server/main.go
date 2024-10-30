package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/eeekcct/lambda-golang-discord-bot/discord/handler"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		headers := ConvertHTTPHeaders(r.Header)

		status, responseBody := handler.HandleRequest(context.Background(), string(body), headers)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(responseBody))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

// ConvertHTTPHeaders normalizes http.Header to map[string]string
func ConvertHTTPHeaders(h http.Header) map[string]string {
	normalized := make(map[string]string)
	for key, values := range h {
		normalized[strings.ToLower(key)] = strings.Join(values, ",")
	}
	return normalized
}
