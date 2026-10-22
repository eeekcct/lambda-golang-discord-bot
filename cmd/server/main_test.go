package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func TestInteractionHandler(t *testing.T) {
	// Disposable test keys only; these requests never contact Discord or AWS.
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DISCORD_PUBLIC_KEY", hex.EncodeToString(publicKey))
	const ping = " { \"type\": 1, \"extra\": \"こんにちは\" }\n"
	const command = `{"type":2,"data":{"name":"ping"}}`
	atLimit := ping + strings.Repeat(" ", maxInteractionBodyBytes-len(ping))

	cases := []struct {
		name          string
		body          io.Reader
		contentLength int64
		signedBody    string
		wantStatus    int
		wantBody      string
	}{
		{"signed PING", strings.NewReader(ping), int64(len(ping)), ping, http.StatusOK, `{"type": 1}`},
		{"unknown length PING", strings.NewReader(ping), -1, ping, http.StatusOK, `{"type": 1}`},
		{"fragmented PING", iotest.OneByteReader(strings.NewReader(ping)), int64(len(ping)), ping, http.StatusOK, `{"type": 1}`},
		{"signed ping command", strings.NewReader(command), int64(len(command)), command, http.StatusOK, `{"type":4,"data":{"content":"pong!"}}`},
		{"unsigned", strings.NewReader(ping), int64(len(ping)), "", http.StatusUnauthorized, "Unauthorized"},
		{"tampered body", strings.NewReader(ping + " "), int64(len(ping) + 1), ping, http.StatusUnauthorized, "Unauthorized"},
		{"exact size limit", strings.NewReader(atLimit), maxInteractionBodyBytes, atLimit, http.StatusOK, `{"type": 1}`},
		{"unknown exact size limit", strings.NewReader(atLimit), -1, atLimit, http.StatusOK, `{"type": 1}`},
		{"known oversized", &unreadableBody{t}, maxInteractionBodyBytes + 1, "", http.StatusRequestEntityTooLarge, "Request body too large\n"},
		{"huge declared length", &unreadableBody{t}, 1 << 62, "", http.StatusRequestEntityTooLarge, "Request body too large\n"},
		{"unknown oversized", io.LimitReader(spaceReader{}, maxInteractionBodyBytes+1), -1, "", http.StatusRequestEntityTooLarge, "Request body too large\n"},
		{"understated length oversized", spaceReader{}, 1, "", http.StatusRequestEntityTooLarge, "Request body too large\n"},
		{"truncated signed body", io.MultiReader(strings.NewReader(ping), iotest.ErrReader(io.ErrUnexpectedEOF)), int64(len(ping) + 1), ping, http.StatusBadRequest, "Unable to read request body\n"},
		{"body read error", iotest.ErrReader(errors.New("read failed")), -1, "", http.StatusBadRequest, "Unable to read request body\n"},
	}
	server := newServer(":9000")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := &countingReader{Reader: tc.body}
			r := httptest.NewRequest(http.MethodPost, "/", body)
			r.ContentLength = tc.contentLength
			if tc.contentLength < 0 {
				r.TransferEncoding = []string{"chunked"}
			}
			if tc.signedBody != "" {
				const timestamp = "1792692000"
				r.Header.Set("X-Signature-Timestamp", timestamp)
				r.Header.Set("X-Signature-Ed25519", hex.EncodeToString(ed25519.Sign(privateKey, []byte(timestamp+tc.signedBody))))
			}
			w := httptest.NewRecorder()
			server.Handler.ServeHTTP(w, r)
			if body.n > maxInteractionBodyBytes+1 {
				t.Fatalf("read %d bytes, want at most limit + 1", body.n)
			}
			if tc.contentLength > maxInteractionBodyBytes && body.n != 0 {
				t.Fatalf("read %d bytes from a declared oversize body", body.n)
			}
			if w.Code != tc.wantStatus || w.Body.String() != tc.wantBody {
				t.Fatalf("response = %d %q, want %d %q", w.Code, w.Body.String(), tc.wantStatus, tc.wantBody)
			}
			if tc.wantStatus == http.StatusOK && w.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

type countingReader struct {
	io.Reader
	n int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.n += n
	return n, err
}

// A declared oversize body must be rejected without reading or allocating it.
type unreadableBody struct{ t *testing.T }

func (r *unreadableBody) Read([]byte) (int, error) {
	r.t.Error("unexpected body read")
	return 0, errors.New("unexpected body read")
}

// Generate over-limit input without allocating an over-limit payload.
type spaceReader struct{}

func (spaceReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

func TestServerTimeouts(t *testing.T) {
	s := newServer(":9000")
	if s.Addr != ":9000" || s.Handler == nil {
		t.Fatalf("unexpected server configuration: %+v", s)
	}
	if s.ReadHeaderTimeout != 5*time.Second || s.ReadTimeout != 10*time.Second ||
		s.WriteTimeout != 10*time.Second || s.IdleTimeout != 60*time.Second {
		t.Fatalf("unexpected server timeouts: %+v", s)
	}
}
