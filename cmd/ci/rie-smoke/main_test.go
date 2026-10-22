package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
)

func response(status int, body string, outer int) []byte {
	encoded, _ := json.Marshal(map[string]any{"statusCode": status, "body": body})
	return append(encoded, []byte(fmt.Sprintf("\n%d", outer))...)
}

func TestValidateResponse(t *testing.T) {
	ping := smokeCase{wantStatus: 200, wantBody: `{"type":1}`, jsonBody: true}
	unauthorized := smokeCase{wantStatus: 401, wantBody: "Unauthorized"}
	for _, tt := range []struct {
		name    string
		raw     []byte
		want    smokeCase
		wantErr bool
	}{
		{"unsigned", response(401, "Unauthorized", 200), unauthorized, false},
		{"signed PING", response(200, `{"type":1}`, 200), ping, false},
		{"handler error despite HTTP 200", response(500, "Error", 200), ping, true},
		{"wrong JSON body", response(200, `{"type":5}`, 200), ping, true},
		{"wrong text body", response(401, "Error", 200), unauthorized, true},
		{"outer HTTP failure", response(200, `{"type":1}`, 500), ping, true},
		{"runtime error payload", []byte("{\"errorMessage\":\"handler failed\"}\n200"), ping, true},
		{"malformed body", response(200, "{", 200), ping, true},
		{"malformed response", []byte("{\n200"), ping, true},
		{"missing HTTP status", []byte("{}"), ping, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateResponse(tt.raw, tt.want); (err != nil) != tt.wantErr {
				t.Fatalf("validateResponse() error = %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestFixtures(t *testing.T) {
	publicHex, cases, err := fixtures()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := hex.DecodeString(publicHex)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		t.Fatalf("invalid test public key: %v", err)
	}
	if len(cases) != 3 || len(cases[0].event.Headers) != 0 || cases[0].wantStatus != 401 {
		t.Fatal("missing unsigned rejection case")
	}
	for _, tc := range cases[1:] {
		t.Run(tc.name, func(t *testing.T) {
			var interaction struct {
				Type int                   `json:"type"`
				Data struct{ Name string } `json:"data"`
			}
			if err := json.Unmarshal([]byte(tc.event.Body), &interaction); err != nil {
				t.Fatal(err)
			}
			if interaction.Type != 1 && !(interaction.Type == 2 && interaction.Data.Name == "ping") {
				t.Fatal("fixture must not invoke downstream AWS or Discord services")
			}
			signature, err := hex.DecodeString(tc.event.Headers["x-signature-ed25519"])
			if err != nil {
				t.Fatal(err)
			}
			message := []byte(tc.event.Headers["x-signature-timestamp"] + tc.event.Body)
			if !ed25519.Verify(publicKey, message, signature) {
				t.Fatal("signature does not verify")
			}
			if ed25519.Verify(publicKey, append(message, '!'), signature) {
				t.Fatal("signature must reject a modified message")
			}
			if err := validateResponse(response(tc.wantStatus, tc.wantBody, 200), tc); err != nil {
				t.Fatal(err)
			}
		})
	}
}
