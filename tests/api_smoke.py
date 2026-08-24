import json
import os
import urllib.request

BASE = os.environ.get("E2E_BASE", "http://localhost:31871")


def get(path: str):
    with urllib.request.urlopen(BASE + path, timeout=8) as r:
        body = r.read()
        ct = r.headers.get("Content-Type", "")
        if "json" in ct:
            return json.loads(body), r.status
        return body.decode(), r.status


def main():
    h, _ = get("/health")
    assert h.get("status") == "ok", h
    q, _ = get("/api/v1/query?query=node_cpu_usage")
    assert q.get("status") in ("success", "error"), q
    print("smoke ok", h, q.get("status"))


if __name__ == "__main__":
    main()
