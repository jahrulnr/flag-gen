#!/usr/bin/env python3
import base64
import json
import os
import shutil
import socket
import struct
import subprocess
import tempfile
import time
import urllib.parse
import urllib.request


APP_URL = os.environ.get("APP_URL", "http://localhost:30405/app")
API_BASE_URL = os.environ.get("API_BASE_URL", "http://localhost:6123/v1")
PROJECT_KEY = os.environ.get("PROJECT_KEY", "development")
PROJECT_ID = os.environ.get("PROJECT_ID", "")
ENVIRONMENT_KEY = os.environ.get("ENVIRONMENT_KEY", "development")
OUT_DIR = os.environ.get("BROWSER_E2E_OUT", "dogfood-output/screenshots")
CHROME = os.environ.get("CHROME_BIN") or shutil.which("google-chrome") or shutil.which("chromium") or shutil.which("chromium-browser")
DEBUG_PORT = int(os.environ.get("CHROME_DEBUG_PORT", "9223"))


class CDP:
    def __init__(self, ws_url):
        parsed = urllib.parse.urlparse(ws_url)
        self.host = parsed.hostname or "localhost"
        self.port = parsed.port or 80
        self.path = parsed.path + (("?" + parsed.query) if parsed.query else "")
        self.sock = socket.create_connection((self.host, self.port), timeout=10)
        self.next_id = 1
        self._handshake()

    def _handshake(self):
        key = base64.b64encode(os.urandom(16)).decode("ascii")
        request = "\r\n".join(
            [
                f"GET {self.path} HTTP/1.1",
                f"Host: {self.host}:{self.port}",
                "Upgrade: websocket",
                "Connection: Upgrade",
                f"Sec-WebSocket-Key: {key}",
                "Sec-WebSocket-Version: 13",
                "\r\n",
            ]
        )
        self.sock.sendall(request.encode("ascii"))
        response = b""
        while b"\r\n\r\n" not in response:
            response += self.sock.recv(4096)
        if not response.startswith(b"HTTP/1.1 101"):
            raise RuntimeError(response.decode("iso-8859-1", "replace"))

    def close(self):
        self.sock.close()

    def command(self, method, params=None):
        msg_id = self.next_id
        self.next_id += 1
        self._send(json.dumps({"id": msg_id, "method": method, "params": params or {}}).encode("utf-8"))
        while True:
            msg = json.loads(self._recv().decode("utf-8"))
            if msg.get("id") == msg_id:
                if "error" in msg:
                    raise RuntimeError(msg["error"])
                return msg.get("result", {})

    def _send(self, payload):
        header = bytearray([0x81])
        length = len(payload)
        if length < 126:
            header.append(0x80 | length)
        elif length < 65536:
            header.append(0x80 | 126)
            header.extend(struct.pack("!H", length))
        else:
            header.append(0x80 | 127)
            header.extend(struct.pack("!Q", length))
        mask = os.urandom(4)
        header.extend(mask)
        masked = bytes(payload[i] ^ mask[i % 4] for i in range(length))
        self.sock.sendall(bytes(header) + masked)

    def _recv(self):
        b1, b2 = self._read(2)
        length = b2 & 0x7F
        if length == 126:
            length = struct.unpack("!H", self._read(2))[0]
        elif length == 127:
            length = struct.unpack("!Q", self._read(8))[0]
        if b2 & 0x80:
            mask = self._read(4)
            payload = self._read(length)
            return bytes(payload[i] ^ mask[i % 4] for i in range(length))
        return self._read(length)

    def _read(self, n):
        out = b""
        while len(out) < n:
            chunk = self.sock.recv(n - len(out))
            if not chunk:
                raise RuntimeError("chrome websocket closed")
            out += chunk
        return out


def resolve_project_id():
    if PROJECT_ID:
        return PROJECT_ID
    with urllib.request.urlopen(f"{API_BASE_URL.rstrip('/')}/projects", timeout=10) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    for project in data.get("projects", []):
        if project.get("Key") == PROJECT_KEY or project.get("key") == PROJECT_KEY:
            return project.get("ID") or project.get("id")
    raise RuntimeError(f"project key not found: {PROJECT_KEY}")


def wait_for_debugger():
    list_url = f"http://127.0.0.1:{DEBUG_PORT}/json/list"
    for _ in range(80):
        try:
            with urllib.request.urlopen(list_url, timeout=1) as resp:
                targets = json.loads(resp.read().decode("utf-8"))
            for target in targets:
                if target.get("type") == "page":
                    return target["webSocketDebuggerUrl"]
        except Exception:
            time.sleep(0.1)
    raise RuntimeError("chrome remote debugger did not start")


def eval_js(cdp, expression, await_promise=False):
    return cdp.command(
        "Runtime.evaluate",
        {"expression": expression, "awaitPromise": await_promise, "returnByValue": True},
    )


def wait_for(cdp, expression):
    script = f"""
new Promise((resolve, reject) => {{
  const started = Date.now();
  const tick = () => {{
    if ({expression}) return resolve(true);
    if (Date.now() - started > 10000) return reject(new Error('timeout waiting for {expression}'));
    setTimeout(tick, 100);
  }};
  tick();
}})
"""
    eval_js(cdp, script, True)


def screenshot(cdp, path):
    data = cdp.command("Page.captureScreenshot", {"format": "png", "captureBeyondViewport": True})["data"]
    with open(path, "wb") as fh:
        fh.write(base64.b64decode(data))


def navigate(cdp, url):
    cdp.command("Page.navigate", {"url": url})
    wait_for(cdp, "document.readyState === 'complete'")


def main():
    if not CHROME:
        raise RuntimeError("Chrome binary not found")
    os.makedirs(OUT_DIR, exist_ok=True)
    project_id = resolve_project_id()
    app_root = APP_URL.rstrip("/")
    profile = tempfile.mkdtemp(prefix="flag-gen-browser-e2e-")
    proc = subprocess.Popen(
        [
            CHROME,
            "--headless=new",
            "--no-sandbox",
            "--disable-gpu",
            "--window-size=1440,1100",
            f"--remote-debugging-port={DEBUG_PORT}",
            f"--user-data-dir={profile}",
            "about:blank",
        ],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    cdp = None
    try:
        cdp = CDP(wait_for_debugger())
        cdp.command("Page.enable")
        cdp.command("Runtime.enable")
        navigate(cdp, f"{app_root}/login")
        eval_js(cdp, "localStorage.setItem('flag-gen-dev-session', 'dev')")

        navigate(cdp, f"{app_root}/dashboard/wiki/developer")
        wait_for(cdp, "document.body.innerText.includes('Developer Wiki') && document.body.innerText.includes('Angular / Browser')")
        screenshot(cdp, os.path.join(OUT_DIR, "browser-developer-wiki.png"))

        navigate(cdp, f"{app_root}/dashboard/projects/{project_id}/environments/{ENVIRONMENT_KEY}/flags")
        wait_for(cdp, "document.body.innerText.includes('Flags') && Array.from(document.querySelectorAll('button')).some(b => b.textContent.includes('Evaluate'))")
        eval_js(cdp, "Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Evaluate')).click()")
        wait_for(cdp, "document.body.innerText.includes('Run evaluation')")
        screenshot(cdp, os.path.join(OUT_DIR, "browser-flags-evaluate-panel.png"))

        eval_js(cdp, "Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Run evaluation')).click()")
        wait_for(cdp, "document.body.innerText.includes('Reason') && document.body.innerText.includes('Source')")
        screenshot(cdp, os.path.join(OUT_DIR, "browser-flags-evaluate-result.png"))
        print(f"ok: browser e2e screenshots written to {OUT_DIR}")
    finally:
        if cdp:
            cdp.close()
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        shutil.rmtree(profile, ignore_errors=True)


if __name__ == "__main__":
    main()
