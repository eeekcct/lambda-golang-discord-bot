package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eeekcct/lambda-golang-discord-bot/discord/config"
	"github.com/eeekcct/lambda-golang-discord-bot/discord/handler"
	"github.com/joho/godotenv"
)

// maxInteractionBodyBytes is an application limit, not a Discord protocol limit.
// 1 MiB leaves headroom for the supported commands and their JSON metadata while
// bounding unauthenticated body reads, including requests with unknown length.
const maxInteractionBodyBytes = 1 << 20

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(newServer("0.0.0.0:" + port).ListenAndServe())
}

func newServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleInteraction)
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func handleInteraction(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > maxInteractionBodyBytes {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxInteractionBodyBytes)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
		}
		return
	}

	headers := ConvertHTTPHeaders(r.Header)
	conf := config.NewConfig()
	h := handler.NewHandler(conf)

	// Keep the original bytes for Discord's signature verification. The existing
	// background context also lets deferred command work outlive this response.
	status, responseBody := h.HandleRequest(context.Background(), string(body), headers, h.HandleServer)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(responseBody))
}

// ConvertHTTPHeaders normalizes http.Header to map[string]string
func ConvertHTTPHeaders(h http.Header) map[string]string {
	normalized := make(map[string]string)
	for key, values := range h {
		normalized[strings.ToLower(key)] = strings.Join(values, ",")
	}
	return normalized
}
