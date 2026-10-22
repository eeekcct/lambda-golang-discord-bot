"""Exercise the actual Lambda images without AWS credentials or network access."""

import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import time
import uuid


def command(args, *, data=None, timeout=30, check=True):
    result = subprocess.run(args, input=data, capture_output=True, timeout=timeout)
    if check and result.returncode:
        raise RuntimeError(f"{args[0]} failed: {result.stderr.decode(errors='replace')}")
    return result


def fixtures(directory):
    """Generate a disposable signing key, never a real Discord credential."""
    key = directory / "test-key.pem"
    command(["openssl", "genpkey", "-algorithm", "ED25519", "-out", str(key)])
    public_der = command([
        "openssl", "pkey", "-in", str(key), "-pubout", "-outform", "DER",
    ]).stdout
    prefix = bytes.fromhex("302a300506032b6570032100")
    if len(public_der) != 44 or not public_der.startswith(prefix):
        raise RuntimeError("Unexpected Ed25519 public-key encoding")
    public_key = public_der[len(prefix):].hex()
    cases = [("unsigned request", {"body": "{}", "headers": {}}, 401, "Unauthorized")]
    timestamp = str(int(time.time()))
    for name, body, response in [
        ("Discord protocol PING", '{"type":1}', {"type": 1}),
        ("Discord /ping command", '{"type":2,"data":{"name":"ping"}}',
         {"type": 4, "data": {"content": "pong!"}}),
    ]:
        payload = directory / "payload"
        payload.write_bytes((timestamp + body).encode())
        signature = command([
            "openssl", "pkeyutl", "-sign", "-rawin", "-inkey", str(key), "-in", str(payload),
        ]).stdout.hex()
        event = {"version": "2.0", "body": body, "headers": {
            "x-signature-ed25519": signature, "x-signature-timestamp": timestamp,
        }}
        cases.append((name, event, 200, response))
    return public_key, cases


def validate_response(raw, expected_status, expected_body):
    # RIE can return HTTP 200 while the Lambda event reports a handler failure.
    body, http_status = raw.rsplit(b"\n", 1)
    if http_status != b"200":
        raise AssertionError(f"Unexpected RIE HTTP status: {http_status!r}")
    response = json.loads(body)
    if response.get("statusCode") != expected_status:
        raise AssertionError(f"Unexpected Lambda status: {response!r}")
    actual_body = response.get("body")
    if isinstance(expected_body, dict):
        actual_body = json.loads(actual_body)
    if actual_body != expected_body:
        raise AssertionError(f"Unexpected Lambda body: {response!r}")


def wait_for_rie(container):
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        running = command([
            "docker", "inspect", "--format", "{{.State.Running}}", container,
        ], timeout=5).stdout.strip()
        if running != b"true":
            raise RuntimeError("Lambda container exited before RIE became ready")
        ready = command([
            "docker", "exec", container, "/usr/bin/curl", "--silent", "--show-error",
            "--max-time", "1", "--output", "/dev/null", "http://127.0.0.1:8080/",
        ], timeout=5, check=False)
        if ready.returncode == 0:
            return
        time.sleep(1)
    raise RuntimeError("RIE did not become ready within 60 seconds")


def smoke(image, target):
    config = json.loads(command([
        "docker", "image", "inspect", "--format", "{{json .Config}}", image,
    ]).stdout)
    expected_entrypoint = ["/functions/main"] if target == "aws" else ["/usr/local/bin/aws-lambda-rie"]
    if config["Entrypoint"] != expected_entrypoint:
        raise AssertionError(f"Unexpected {target} entrypoint: {config['Entrypoint']}")
    if target == "local" and config["Cmd"] != ["/functions/main"]:
        raise AssertionError(f"Unexpected local command: {config['Cmd']}")

    container = "lambda-smoke-" + uuid.uuid4().hex
    with tempfile.TemporaryDirectory(prefix="lambda-smoke-") as directory:
        public_key, cases = fixtures(Path(directory))
        try:
            args = [
                "docker", "run", "--detach", "--name", container,
                "--label", "lambda-bot-ci=true", "--platform", "linux/amd64",
                "--network", "none", "--read-only", "--tmpfs", "/tmp:rw,nosuid,size=64m",
                "--env", "DISCORD_PUBLIC_KEY=" + public_key,
                "--env", "AWS_EC2_METADATA_DISABLED=true",
            ]
            # AWS's deployment target has a direct binary entrypoint; wrap it
            # with the RIE already in the image only for local invocation.
            if target == "aws":
                args += ["--entrypoint", "/usr/local/bin/aws-lambda-rie", image, "/functions/main"]
            else:
                args += [image]
            command(args)
            wait_for_rie(container)
            command(["docker", "exec", container, "/bin/sh", "-c",
                     "test -x /functions/main && test -x /functions/bedrock"])
            for name, event, status, body in cases:
                result = command([
                    "docker", "exec", "--interactive", container, "/usr/bin/curl",
                    "--silent", "--show-error", "--fail-with-body", "--max-time", "20",
                    "--header", "Content-Type: application/json", "--data-binary", "@-",
                    "--write-out", "\n%{http_code}",
                    "http://127.0.0.1:8080/2015-03-31/functions/function/invocations",
                ], data=json.dumps(event).encode(), timeout=25)
                validate_response(result.stdout, status, body)
                print(f"PASS {target}: {name}", flush=True)
        except Exception:
            logs = command(["docker", "logs", container], timeout=10, check=False)
            print((logs.stdout + logs.stderr).decode(errors="replace"), flush=True)
            raise
        finally:
            # Also attempt removal if docker run had an uncertain failure.
            removed = command(["docker", "rm", "--force", container], timeout=20, check=False)
            if removed.returncode:
                print(removed.stderr.decode(errors="replace"), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", required=True)
    parser.add_argument("--target", choices=["aws", "local"], required=True)
    args = parser.parse_args()
    smoke(args.image, args.target)
