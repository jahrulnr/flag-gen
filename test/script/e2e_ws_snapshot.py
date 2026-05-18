#!/usr/bin/env python3
import base64
import hashlib
import json
import os
import socket
import ssl
import struct
import sys
import urllib.parse
import urllib.request


BASE_WS_URL = os.environ.get("BASE_WS_URL", "ws://localhost:6123/v1")
BASE_URL = os.environ.get("BASE_URL", "http://localhost:6123/v1")
PROJECT_KEY = os.environ.get("PROJECT_KEY", "development")
PROJECT_ID = os.environ.get("PROJECT_ID", "")
ENVIRONMENT_KEY = os.environ.get("ENVIRONMENT_KEY", "development")
BROADCAST_TOKEN = os.environ.get("BROADCAST_TOKEN", "")
API_KEY = os.environ.get("API_KEY", "")
TIMEOUT_SECONDS = float(os.environ.get("WS_TIMEOUT_SECONDS", "5"))


def build_url():
    base = BASE_WS_URL.rstrip("/")
    qs = {"projectId": PROJECT_ID, "environmentKey": ENVIRONMENT_KEY}
    if BROADCAST_TOKEN:
        qs["token"] = BROADCAST_TOKEN
    return f"{base}/broadcast/flags?{urllib.parse.urlencode(qs)}"


def resolve_project_id():
    if PROJECT_ID:
        return PROJECT_ID
    with urllib.request.urlopen(f"{BASE_URL.rstrip('/')}/projects", timeout=TIMEOUT_SECONDS) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    for project in data.get("projects", []):
        if project.get("Key") == PROJECT_KEY or project.get("key") == PROJECT_KEY:
            return project.get("ID") or project.get("id")
    raise RuntimeError(f"project key not found: {PROJECT_KEY}")


def read_exact(sock, n):
    chunks = []
    remaining = n
    while remaining > 0:
        chunk = sock.recv(remaining)
        if not chunk:
            raise RuntimeError("connection closed while reading websocket frame")
        chunks.append(chunk)
        remaining -= len(chunk)
    return b"".join(chunks)


def read_frame(sock):
    b1, b2 = read_exact(sock, 2)
    opcode = b1 & 0x0F
    masked = (b2 & 0x80) != 0
    length = b2 & 0x7F
    if length == 126:
        length = struct.unpack("!H", read_exact(sock, 2))[0]
    elif length == 127:
        length = struct.unpack("!Q", read_exact(sock, 8))[0]
    mask = read_exact(sock, 4) if masked else b""
    payload = read_exact(sock, length)
    if masked:
        payload = bytes(payload[i] ^ mask[i % 4] for i in range(length))
    if opcode == 8:
        raise RuntimeError("websocket closed before snapshot")
    if opcode != 1:
        raise RuntimeError(f"expected text frame, got opcode={opcode}")
    return payload.decode("utf-8")


def connect(url):
    parsed = urllib.parse.urlparse(url)
    port = parsed.port or (443 if parsed.scheme == "wss" else 80)
    host = parsed.hostname or "localhost"
    path = parsed.path or "/"
    if parsed.query:
        path += "?" + parsed.query

    raw = socket.create_connection((host, port), timeout=TIMEOUT_SECONDS)
    raw.settimeout(TIMEOUT_SECONDS)
    sock = ssl.create_default_context().wrap_socket(raw, server_hostname=host) if parsed.scheme == "wss" else raw

    key = base64.b64encode(os.urandom(16)).decode("ascii")
    headers = [
        f"GET {path} HTTP/1.1",
        f"Host: {host}:{port}",
        "Upgrade: websocket",
        "Connection: Upgrade",
        f"Sec-WebSocket-Key: {key}",
        "Sec-WebSocket-Version: 13",
    ]
    if API_KEY:
        headers.append(f"Authorization: Bearer {API_KEY}")
    request = "\r\n".join(headers) + "\r\n\r\n"
    sock.sendall(request.encode("ascii"))

    response = b""
    while b"\r\n\r\n" not in response:
        chunk = sock.recv(4096)
        if not chunk:
            raise RuntimeError("connection closed during websocket handshake")
        response += chunk
    header_text = response.decode("iso-8859-1")
    if not header_text.startswith("HTTP/1.1 101"):
        raise RuntimeError(f"websocket handshake failed:\n{header_text}")
    accept = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode("ascii")).digest()).decode("ascii")
    if accept not in header_text:
        raise RuntimeError("websocket accept header mismatch")
    return sock


def main():
    global PROJECT_ID
    PROJECT_ID = resolve_project_id()
    url = build_url()
    print(f"ws={url}")
    sock = connect(url)
    try:
        message = read_frame(sock)
    finally:
        sock.close()

    data = json.loads(message)
    assert data["type"] == "flags.snapshot", data
    assert data["projectId"] == PROJECT_ID, data
    assert data["environmentKey"] == ENVIRONMENT_KEY, data
    assert isinstance(data.get("scopeVersion"), int), data
    assert "occurredAt" in data, data
    assert isinstance(data.get("flags"), list), data
    if data["flags"]:
        flag = data["flags"][0]
        for key in ("id", "key", "enabled", "clientSide", "value", "rules", "version"):
            assert key in flag, flag
    print("ok: websocket snapshot contract e2e passed")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"failed: {exc}", file=sys.stderr)
        sys.exit(1)
