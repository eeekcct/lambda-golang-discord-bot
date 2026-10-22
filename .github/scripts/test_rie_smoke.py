import json
from pathlib import Path
import tempfile
import unittest

from rie_smoke import command, fixtures, validate_response


def response(status, body, outer=200):
    if isinstance(body, dict):
        body = json.dumps(body)
    return json.dumps({"statusCode": status, "body": body}).encode() + f"\n{outer}".encode()


class ResponseTests(unittest.TestCase):
    def test_accepts_expected_unauthorized_and_ping(self):
        validate_response(response(401, "Unauthorized"), 401, "Unauthorized")
        validate_response(response(200, {"type": 1}), 200, {"type": 1})

    def test_rejects_handler_error_despite_outer_http_200(self):
        with self.assertRaises(AssertionError):
            validate_response(response(500, "Error"), 200, {"type": 1})

    def test_rejects_wrong_body_or_outer_status(self):
        for raw in (response(200, {"type": 5}), response(200, {"type": 1}, outer=500)):
            with self.subTest(raw=raw), self.assertRaises(AssertionError):
                validate_response(raw, 200, {"type": 1})

    def test_rejects_runtime_error_payload(self):
        with self.assertRaises(AssertionError):
            validate_response(b'{"errorMessage":"handler failed"}\n200', 200, {"type": 1})


class FixtureTests(unittest.TestCase):
    def test_ephemeral_signatures_verify_and_only_safe_cases_are_generated(self):
        with tempfile.TemporaryDirectory() as directory:
            directory = Path(directory)
            public_key, cases = fixtures(directory)
            self.assertEqual(len(public_key), 64)
            self.assertEqual(len(cases), 3)
            self.assertEqual(cases[0][1]["headers"], {})
            public_der = directory / "public.der"
            public_der.write_bytes(bytes.fromhex("302a300506032b6570032100" + public_key))
            for _, event, status, _ in cases[1:]:
                self.assertEqual(status, 200)
                interaction = json.loads(event["body"])
                self.assertIn(interaction["type"], (1, 2))
                if interaction["type"] == 2:
                    self.assertEqual(interaction["data"]["name"], "ping")
                payload = directory / "verify-payload"
                payload.write_bytes((event["headers"]["x-signature-timestamp"] + event["body"]).encode())
                signature = directory / "signature"
                signature.write_bytes(bytes.fromhex(event["headers"]["x-signature-ed25519"]))
                command(["openssl", "pkeyutl", "-verify", "-rawin", "-pubin",
                         "-inkey", str(public_der), "-keyform", "DER", "-sigfile", str(signature),
                         "-in", str(payload)])


if __name__ == "__main__":
    unittest.main()
