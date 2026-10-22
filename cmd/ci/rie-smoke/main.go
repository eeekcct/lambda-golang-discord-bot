// rie-smoke exercises the actual Lambda images without AWS credentials or networking.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"time"
)

type event struct {
	Version string            `json:"version"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

type smokeCase struct {
	name       string
	event      event
	wantStatus int
	wantBody   string
	jsonBody   bool
}

func fixtures() (string, []smokeCase, error) {
	// This disposable test key stays in memory; it is not a Discord credential.
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, err
	}
	cases := []smokeCase{
		{"unsigned request", event{"2.0", "{}", map[string]string{}}, 401, "Unauthorized", false},
		{"Discord protocol PING", event{Body: `{"type":1}`}, 200, `{"type":1}`, true},
		{"Discord /ping command", event{Body: `{"type":2,"data":{"name":"ping"}}`}, 200,
			`{"type":4,"data":{"content":"pong!"}}`, true},
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	for i := 1; i < len(cases); i++ {
		cases[i].event.Version = "2.0"
		cases[i].event.Headers = map[string]string{
			"x-signature-timestamp": timestamp,
			"x-signature-ed25519": hex.EncodeToString(ed25519.Sign(privateKey,
				[]byte(timestamp+cases[i].event.Body))),
		}
	}
	return hex.EncodeToString(publicKey), cases, nil
}

func command(timeout time.Duration, input []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s failed: %w: %s", args[0], err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func printContainerLogs(container string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "logs", container)
	cmd.WaitDelay = 2 * time.Second
	logs, err := cmd.CombinedOutput()
	fmt.Fprint(os.Stderr, string(logs))
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading container logs:", err)
	}
}

func validateResponse(raw []byte, tc smokeCase) error {
	// HTTP 200 from RIE does not imply that the Lambda handler succeeded.
	separator := bytes.LastIndexByte(raw, '\n')
	if separator < 0 || string(raw[separator+1:]) != "200" {
		return fmt.Errorf("unexpected RIE HTTP status in %q", raw)
	}
	var response struct {
		StatusCode int    `json:"statusCode"`
		Body       string `json:"body"`
	}
	if err := json.Unmarshal(raw[:separator], &response); err != nil {
		return fmt.Errorf("invalid Lambda response: %w", err)
	}
	if response.StatusCode != tc.wantStatus {
		return fmt.Errorf("Lambda status = %d, want %d: %s", response.StatusCode, tc.wantStatus, response.Body)
	}
	if tc.jsonBody {
		var got, want any
		if err := json.Unmarshal([]byte(response.Body), &got); err != nil {
			return fmt.Errorf("invalid Lambda response body: %w", err)
		}
		if err := json.Unmarshal([]byte(tc.wantBody), &want); err != nil {
			return fmt.Errorf("invalid expected body: %w", err)
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("Lambda body = %s, want %s", response.Body, tc.wantBody)
		}
	} else if response.Body != tc.wantBody {
		return fmt.Errorf("Lambda body = %q, want %q", response.Body, tc.wantBody)
	}
	return nil
}

func waitForRIE(container string) error {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		running, err := command(5*time.Second, nil,
			"docker", "inspect", "--format", "{{.State.Running}}", container)
		if err != nil {
			return err
		}
		if string(bytes.TrimSpace(running)) != "true" {
			return fmt.Errorf("Lambda container exited before RIE became ready")
		}
		_, err = command(5*time.Second, nil, "docker", "exec", container,
			"/usr/bin/curl", "--silent", "--show-error", "--max-time", "1",
			"--output", "/dev/null", "http://127.0.0.1:8080/")
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("RIE did not become ready within 60 seconds")
}

func smoke(image, target string) (err error) {
	configJSON, err := command(30*time.Second, nil,
		"docker", "image", "inspect", "--format", "{{json .Config}}", image)
	if err != nil {
		return err
	}
	var config struct{ Entrypoint, Cmd []string }
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return err
	}
	expectedEntrypoint := []string{"/functions/main"}
	if target == "local" {
		expectedEntrypoint = []string{"/usr/local/bin/aws-lambda-rie"}
	}
	if !slices.Equal(config.Entrypoint, expectedEntrypoint) {
		return fmt.Errorf("unexpected %s entrypoint: %v", target, config.Entrypoint)
	}
	if target == "local" && !slices.Equal(config.Cmd, []string{"/functions/main"}) {
		return fmt.Errorf("unexpected local command: %v", config.Cmd)
	}
	publicKey, cases, err := fixtures()
	if err != nil {
		return err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	container := "lambda-smoke-" + hex.EncodeToString(id[:])
	defer func() {
		if err != nil {
			printContainerLogs(container)
		}
		// Try removal even if docker run failed after creating the container.
		if _, cleanupErr := command(20*time.Second, nil, "docker", "rm", "--force", container); cleanupErr != nil {
			fmt.Fprintln(os.Stderr, cleanupErr)
			if err == nil {
				err = cleanupErr
			}
		}
	}()
	args := []string{"docker", "run", "--detach", "--name", container,
		"--label", "lambda-bot-ci=true", "--platform", "linux/amd64",
		"--network", "none", "--read-only", "--tmpfs", "/tmp:rw,nosuid,size=64m",
		"--env", "DISCORD_PUBLIC_KEY=" + publicKey, "--env", "AWS_EC2_METADATA_DISABLED=true"}
	// Wrap the aws target's direct entrypoint with its bundled RIE only for
	// local invocation; exercise the local target's default entrypoint as-is.
	if target == "aws" {
		args = append(args, "--entrypoint", "/usr/local/bin/aws-lambda-rie", image, "/functions/main")
	} else {
		args = append(args, image)
	}
	if _, err = command(30*time.Second, nil, args...); err != nil {
		return err
	}
	if err = waitForRIE(container); err != nil {
		return err
	}
	if _, err = command(30*time.Second, nil, "docker", "exec", container, "/bin/sh", "-c",
		"test -x /functions/main && test -x /functions/bedrock"); err != nil {
		return err
	}
	for _, tc := range cases {
		payload, marshalErr := json.Marshal(tc.event)
		if marshalErr != nil {
			return marshalErr
		}
		raw, invokeErr := command(25*time.Second, payload, "docker", "exec", "--interactive", container,
			"/usr/bin/curl", "--silent", "--show-error", "--fail-with-body", "--max-time", "20",
			"--header", "Content-Type: application/json", "--data-binary", "@-",
			"--write-out", "\n%{http_code}", "http://127.0.0.1:8080/2015-03-31/functions/function/invocations")
		if invokeErr != nil {
			return fmt.Errorf("%s: %w", tc.name, invokeErr)
		}
		if err = validateResponse(raw, tc); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
		fmt.Printf("PASS %s: %s\n", target, tc.name)
	}
	return nil
}

func main() {
	image := flag.String("image", "", "local Docker image to test")
	target := flag.String("target", "", "Dockerfile target: aws or local")
	flag.Parse()
	if *image == "" || (*target != "aws" && *target != "local") {
		fmt.Fprintln(os.Stderr, "-image and -target (aws or local) are required")
		os.Exit(2)
	}
	if err := smoke(*image, *target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
