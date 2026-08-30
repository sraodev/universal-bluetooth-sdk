import json
import os
from pathlib import Path
from http.server import BaseHTTPRequestHandler, HTTPServer
from tempfile import TemporaryDirectory
from threading import Thread
import unittest

from local_draft import draft, save_draft


class DraftTests(unittest.TestCase):
    def call_server(self, payload, status=200):
        requests = []

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                requests.append((self.path, json.loads(self.rfile.read(int(self.headers["Content-Length"])))))
                self.send_response(status)
                self.end_headers()
                self.wfile.write(payload)

            def log_message(self, *args):
                pass

        server = HTTPServer(("127.0.0.1", 0), Handler)
        server.timeout = 3
        worker = Thread(target=server.handle_request, daemon=True)
        worker.start()
        try:
            return draft("Say hello", "test-local", server.server_port), requests
        finally:
            worker.join(timeout=4)
            server.server_close()

    def test_completed_draft_uses_nonstreaming_api_without_tools(self):
        text, requests = self.call_server(json.dumps({"response": "Hello!", "done": True}).encode())
        self.assertEqual(text, "Hello!")
        self.assertEqual(requests[0][0], "/api/generate")
        self.assertIs(requests[0][1]["stream"], False)
        self.assertNotIn("tools", requests[0][1])

    def test_invalid_responses_are_rejected(self):
        for payload in (b"not json", b"[]", b'{"response":"hi","done":false}', b'{"response":"","done":true}', b'{"response":"\\u001b[2J","done":true}', json.dumps({"response":"x"*1025,"done":True}).encode(), b"x"*65537):
            with self.subTest(payload=payload[:50]), self.assertRaises(ValueError):
                self.call_server(payload)

    def test_http_error(self):
        with self.assertRaisesRegex(ValueError, "HTTP 404"):
            self.call_server(b'{}', 404)

    def test_cloud_and_oversized_inputs_rejected_before_connection(self):
        for prompt,model in (("hi","example-cloud"),("", "local"),("x"*4097,"local")):
            with self.subTest(model=model), self.assertRaises(ValueError):
                draft(prompt,model,1)

    def test_private_file_and_no_overwrite(self):
        with TemporaryDirectory() as directory:
            path = Path(directory) / "draft.txt"
            save_draft(path, "hello")
            self.assertEqual(path.read_text(), "hello\n")
            if os.name == "posix":
                self.assertEqual(path.stat().st_mode & 0o777, 0o600)
            with self.assertRaises(FileExistsError):
                save_draft(path, "replacement")
            self.assertEqual(path.read_text(), "hello\n")


if __name__ == "__main__":
    unittest.main()
