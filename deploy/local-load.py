#!/usr/bin/env python3
"""Bounded loopback-only HTTP measurements; no production targets allowed."""
import argparse
import concurrent.futures
import http.client
import json
import time
from collections import Counter
from pathlib import Path


def measure(port, path, count, workers, pid):
    def usage():
        fields = Path(f"/proc/{pid}/stat").read_text().split()
        return sum(map(int, fields[13:15]))

    def request(_):
        conn = http.client.HTTPConnection("127.0.0.1", port, timeout=10)
        start = time.monotonic()
        try:
            conn.request("GET", path)
            res = conn.getresponse()
            res.read()
            return res.status, (time.monotonic() - start) * 1000
        except (OSError, http.client.HTTPException):
            return "error", (time.monotonic() - start) * 1000
        finally:
            conn.close()

    ticks = usage()
    start = time.monotonic()
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        results = list(pool.map(request, range(count)))
    duration = time.monotonic() - start
    latencies = sorted(ms for _, ms in results)
    memory = {
        key.rstrip(":"): value.strip()
        for line in Path(f"/proc/{pid}/status").read_text().splitlines()
        if line.startswith(("VmRSS:", "VmHWM:", "Threads:"))
        for key, value in [line.split(":", 1)]
    }
    return dict(path=path, requests=count, concurrency=workers,
                seconds=round(duration, 3), rps=round(count / duration, 1),
                p50_ms=round(latencies[len(latencies)//2], 2),
                p95_ms=round(latencies[int(len(latencies)*.95)], 2),
                statuses=dict(Counter(str(status) for status, _ in results)),
                cpu_ticks=usage()-ticks, memory=memory)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--port", type=int, default=33333)
    parser.add_argument("--pid", type=int, required=True)
    parser.add_argument("--count", type=int, default=300)
    parser.add_argument("--workers", type=int, default=8)
    args = parser.parse_args()
    if not 1 <= args.count <= 2000 or not 1 <= args.workers <= 32:
        parser.error("count must be 1..2000 and workers 1..32")
    deadline = time.monotonic() + 10
    while True:
        conn = http.client.HTTPConnection("127.0.0.1", args.port, timeout=1)
        try:
            conn.request("GET", "/healthz")
            res = conn.getresponse()
            res.read()
            if res.status == 200:
                break
        except (OSError, http.client.HTTPException):
            pass
        finally:
            conn.close()
        if time.monotonic() >= deadline:
            raise SystemExit("local server did not become healthy")
        time.sleep(.1)
    for path in ["/healthz", "/css/style.css", "/login", "/register", "/missing-perf-check"]:
        result = measure(args.port, path, args.count, args.workers, args.pid)
        print(json.dumps(result), flush=True)
        if "error" in result["statuses"]:
            raise SystemExit("transport errors invalidate this measurement")
