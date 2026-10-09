#!/usr/bin/env python3
"""Run govulncheck in a bounded cgroup and persist its actual resource usage."""
import datetime
import fcntl
import signal
import json
import os
from pathlib import Path
import subprocess
import sys
import time

root = Path(__file__).resolve().parent.parent
os.chdir(root)
out = root / 'storage/security-scan'
out.mkdir(parents=True, exist_ok=True, mode=0o700)
lock = (out / 'run.lock').open('a')
try:
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
except BlockingIOError:
    raise SystemExit('A security scan is already running')
(root / 'storage/_build-tmp').mkdir(parents=True, exist_ok=True)
unit = f'bufalo-security-scan-{time.time_ns()}.scope'
analyzer = os.environ.get('GOVULNCHECK_BIN', str(root / 'storage/tools/govulncheck'))
limit = int(os.environ.get('BUFALO_SCAN_MEMORY_BYTES', str(1 << 30)))
if not (256 << 20) <= limit <= (2 << 30):
    raise SystemExit('Memory limit must be between 256 MiB and 2 GiB')
resources_path = out / (unit + '.resources.json')
arguments = sys.argv[1:] or ['-scan=package', './...']
command = ['systemd-run', '--user', '--scope', '--quiet', '--unit=' + unit,
           '-p', f'MemoryMax={limit}', '-p', 'MemorySwapMax=0', '-p', 'CPUQuota=100%', '-p', 'TasksMax=64',
           'env', 'TMPDIR=' + str(root / 'storage/_build-tmp'), 'GOMEMLIMIT=512MiB',
           'GOMAXPROCS=2', 'GOFLAGS=-p=1 -trimpath', 'CGO_ENABLED=0',
           'python3', str(root / 'deploy/security-scan-worker.py'), str(resources_path),
           'timeout', '--signal=TERM', '--kill-after=15s', '15m', analyzer,
           *arguments]
started = time.monotonic()
now = lambda: datetime.datetime.now(datetime.timezone.utc).isoformat()
scan_mode = 'module-inventory' if '-scan=module' in arguments else 'binary-symbols' if '-mode=binary' in arguments else ('source-packages' if '-scan=package' in arguments else 'source-symbols')
state = dict(mode=scan_mode, status='running', started_at=now(), finished_at='', memory_peak_bytes=0,
             memory_limit_bytes=limit, cpu_seconds=0, duration_seconds=0, exit_code=None, result='')

def save():
    tmp = out / 'latest.tmp'
    tmp.write_text(json.dumps(state, indent=2) + '\n')
    tmp.chmod(0o600)
    tmp.replace(out / 'latest.json')

def sample():
    result = subprocess.run(['systemctl', '--user', 'show', unit, '-p', 'MemoryPeak', '-p', 'CPUUsageNSec', '-p', 'Result'], capture_output=True, text=True)
    fields = dict(line.split('=', 1) for line in result.stdout.splitlines() if '=' in line)
    peak = fields.get('MemoryPeak', '')
    if peak.isdigit(): state['memory_peak_bytes'] = max(state['memory_peak_bytes'], int(peak))
    cpu = fields.get('CPUUsageNSec', '')
    if cpu.isdigit(): state['cpu_seconds'] = max(state['cpu_seconds'], int(cpu) / 1e9)
    if fields.get('Result'): state['result'] = fields['Result']
    state['duration_seconds'] = round(time.monotonic() - started, 3)

def interrupted(signum, frame):
    raise KeyboardInterrupt
signal.signal(signal.SIGTERM, interrupted)
save()
logpath = out / (unit + '.log')
try:
    with logpath.open('w') as log:
        logpath.chmod(0o600)
        proc = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT)
        while proc.poll() is None:
            sample(); save(); time.sleep(1)
        sample()
        if resources_path.exists():
            resources = json.loads(resources_path.read_text())
            state['memory_peak_bytes'] = max(state['memory_peak_bytes'], resources.get('memory_peak_bytes', 0))
            state['cpu_seconds'] = max(state['cpu_seconds'], resources.get('cpu_seconds', 0))
            resources_path.unlink()
        state['exit_code'] = proc.returncode
        state['status'] = 'passed' if proc.returncode == 0 else 'failed'
        if not state['result'] or state['result'] == 'success': state['result'] = 'success' if proc.returncode == 0 else ('vulnerabilities_found' if proc.returncode == 3 else 'analyzer_failed')
except KeyboardInterrupt:
    subprocess.run(['systemctl', '--user', 'stop', unit], capture_output=True)
    state['status'], state['result'], state['exit_code'] = 'cancelled', 'cancelled', 130
except Exception:
    state['status'], state['result'] = 'failed', 'launcher_failed'
    raise
finally:
    state['finished_at'] = now(); save()
    report = out / (unit + '.json')
    report.write_text(json.dumps(state, indent=2) + '\n'); report.chmod(0o600)
    # Keep five bounded-by-time analyzer reports; never touch unrelated files.
    logs = sorted(out.glob('bufalo-security-scan-*.scope.log'), key=lambda p:p.stat().st_mtime, reverse=True)
    for old in logs[5:]:
        report = old.with_suffix('.json')
        if report.exists(): report.unlink()
        old.unlink()
print(json.dumps(state, indent=2))
sys.exit(state['exit_code'] or (0 if state['status'] == 'passed' else 1))
