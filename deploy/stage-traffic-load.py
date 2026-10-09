#!/usr/bin/env python3
"""Bounded abuse test; isolated loopback server with DDOS_MODE=enforce only."""
import collections
import hashlib
import concurrent.futures
import http.client
import json
import pathlib
import socket
import time
import threading
import uuid

HOST, PORT = '127.0.0.1', 33440
stage = json.loads(pathlib.Path('/tmp/bufalo-production-stage.json').read_text())
assert stage['env']['DB_DATABASE'].endswith('_test'), 'Isolated staging database required'
assert stage['env']['DDOS_MODE'] == 'enforce'
pid = stage['pid']
run = uuid.uuid4().bytes
ip = f'198.18.{run[0]}.{run[1]}'

def request(path='/login', identity=ip):
    conn = http.client.HTTPConnection(HOST, PORT, timeout=10)
    try:
        conn.request('GET', path, headers={'CF-Connecting-IP': identity})
        response = conn.getresponse()
        response.read()
        return response.status, response.getheader('Retry-After')
    finally:
        conn.close()

def memory():
    data = pathlib.Path(f'/proc/{pid}/status').read_text().splitlines()
    return {line.split(':')[0]: int(line.split()[1]) for line in data if line.startswith(('VmRSS:', 'VmHWM:'))}

assert request('/healthz')[0] == 200
before = memory()
normal = []
for _ in range(20):
    normal.append(request()[0])
    time.sleep(.1)
assert normal == [200] * 20, normal
probes = []
stop = threading.Event()
def probe():
    while not stop.is_set():
        started = time.monotonic()
        try:
            status = request('/login', f'198.19.{run[0]}.{run[1]}')[0]
            health = request('/healthz')[0]
            probes.append({'status': status, 'health': health, 'seconds': time.monotonic() - started})
        except Exception as exc:
            probes.append({'error': type(exc).__name__})
        stop.wait(.5)
thread = threading.Thread(target=probe)
thread.start()
start = time.monotonic()
with concurrent.futures.ThreadPoolExecutor(max_workers=32) as pool:
    results = list(pool.map(lambda _: request(), range(10000)))
elapsed = time.monotonic() - start
stop.set()
thread.join(timeout=12)
assert probes and all(p.get("status") == 200 and p.get("health") == 200 for p in probes), probes
counts = collections.Counter(status for status, _ in results)
assert set(counts) <= {200, 429}, counts
assert counts[429] > 9500, counts
assert all(retry and int(retry) >= 1 for status, retry in results if status == 429)
assert request('/healthz')[0] == 200
assert request('/login', f'198.19.{run[0]}.{run[1]}')[0] == 200
# Oversized authentication is rejected immediately, without sending the body.
with socket.create_connection((HOST, PORT), timeout=3) as conn:
    conn.sendall(b'POST /login HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1048576\r\nConnection: close\r\n\r\n')
    assert b' 413 ' in conn.recv(1024).split(b'\r\n')[0]
print(json.dumps({'binary_sha256': hashlib.sha256(pathlib.Path(f'/proc/{pid}/exe').read_bytes()).hexdigest(), 'independent_probes_during_attack': probes, 'requests': 10000, 'workers': 32, 'seconds': elapsed,
                  'requests_per_second': 10000 / elapsed, 'statuses': dict(counts),
                  'memory_before_kib': before, 'memory_after_kib': memory(),
                  'health': 200, 'independent_ip': 200, 'oversized_auth': 413}, indent=2))
