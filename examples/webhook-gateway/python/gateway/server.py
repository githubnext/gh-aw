"""Loopback HTTP ingress and local-only administrative commands."""

from __future__ import annotations

import argparse
import json
import logging
import re
import signal
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from .common import Rejected, encode, require
from .policy import read_policy
from .store import MAX_BODY_BYTES, Store

LOG = logging.getLogger("webhook_gateway")


class ReferenceServer(ThreadingHTTPServer):
    daemon_threads = True

    def handle_error(self, *_):
        LOG.error("ingress_connection_failed")


def handler(store):
    class Handler(BaseHTTPRequestHandler):
        server_version = "WebhookGatewayReference"

        def log_message(self, *_):
            pass

        def setup(self):
            super().setup()
            self.connection.settimeout(5)

        def reply(self, status, result):
            raw = encode(result)
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(raw)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(raw)

        def do_GET(self):
            self.reply(200 if self.path == "/healthz" else 404,
                       {"status": "running"} if self.path == "/healthz" else {"error": "not_found"})

        def do_POST(self):
            try:
                match = re.fullmatch(r"/ingress/([A-Za-z0-9_-]{1,100})", self.path)
                require(match, "not_found", 404)
                require(not self.headers.get("Transfer-Encoding"), "unsupported_transfer_encoding")
                for name in self.headers:
                    require(len(self.headers.get_all(name)) == 1, "duplicate_http_header")
                length = self.headers.get("Content-Length", "")
                require(re.fullmatch(r"[0-9]{1,9}", length), "content_length_required", 411)
                require(int(length) <= MAX_BODY_BYTES, "ingress_body_too_large", 413)
                require(self.headers.get_content_type() == "application/json", "json_content_type_required", 415)
                raw = self.rfile.read(int(length))
                require(len(raw) == int(length), "incomplete_request_body")
                headers = {k.lower(): v for k, v in self.headers.items()}
                result = store.admit(match[1], headers, raw)
                self.reply(200, result)
            except Rejected as error:
                LOG.warning("ingress_rejected code=%s", error.code)
                self.reply(error.status, {"error": error.code})
            except (OSError, TimeoutError):
                LOG.warning("ingress_transport_failed")
                self.close_connection = True
            except Exception:
                LOG.error("ingress_rejected code=unexpected_internal_error")
                self.reply(500, {"error": "unexpected_internal_error"})

    return Handler


def serve(store, port):
    stopped = threading.Event()
    worker_failed = threading.Event()
    server = ReferenceServer(("127.0.0.1", port), handler(store))

    def worker():
        while not stopped.is_set():
            try:
                if not store.work_once():
                    stopped.wait(0.25)
            except Rejected as error:
                LOG.error("worker_stopped code=%s", error.code)
                worker_failed.set()
                stopped.set()
                server.shutdown()
                return
            except Exception:
                # Do not render an exception that may contain HTTP credentials.
                LOG.error("worker_stopped code=unexpected_internal_error")
                worker_failed.set()
                stopped.set()
                server.shutdown()
                return

    def stop(*_):
        stopped.set()
        threading.Thread(target=server.shutdown, daemon=True).start()

    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    thread = threading.Thread(target=worker)
    thread.start()
    LOG.info("listening address=127.0.0.1 port=%d", server.server_address[1])
    try:
        server.serve_forever()
    finally:
        stopped.set()
        server.server_close()
        thread.join()
    require(not worker_failed.is_set(), "dispatch_worker_failed", 503)
    require(not any(row["state"] == "inflight" for row in store.status()), "unresolved_inflight_jobs", 503)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--db", required=True, help="Private durable SQLite path")
    commands = parser.add_subparsers(dest="command", required=True)
    install = commands.add_parser("install-policy", help="Trusted local approval import, never an ingress action")
    install.add_argument("path")
    run = commands.add_parser("serve")
    run.add_argument("--port", type=int, default=8080)
    commands.add_parser("work-once")
    commands.add_parser("status")
    args = parser.parse_args()
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
    try:
        store = Store(args.db)
        if args.command == "install-policy":
            store.install(read_policy(args.path))
            print('{"status":"policy_installed"}')
        elif args.command == "serve":
            serve(store, args.port)
        elif args.command == "work-once":
            print(json.dumps({"worked": store.work_once()}))
        else:
            print(json.dumps(store.status()))
    except Rejected as error:
        LOG.error("operation_failed code=%s", error.code)
        raise SystemExit(1) from None
    except (OSError, KeyError, TypeError, ValueError):
        LOG.error("operation_failed code=invalid_local_configuration")
        raise SystemExit(1) from None


if __name__ == "__main__":
    main()
