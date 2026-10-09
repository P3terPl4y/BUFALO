#!/usr/bin/env python3
"""Capture final cgroup high-water counters before the scope disappears."""
import json
import os
from pathlib import Path
import subprocess
import sys

os.umask(0o077)
result = subprocess.call(sys.argv[2:])
resources = {}
try:
    relative = next(line.split(':', 2)[2] for line in Path('/proc/self/cgroup').read_text().splitlines() if line.startswith('0::'))
    group = Path('/sys/fs/cgroup') / relative.lstrip('/')
    resources['memory_peak_bytes'] = int((group / 'memory.peak').read_text())
    cpu = dict(line.split() for line in (group / 'cpu.stat').read_text().splitlines())
    resources['cpu_seconds'] = int(cpu['usage_usec']) / 1e6
except (OSError, ValueError, StopIteration, KeyError):
    resources['final_counters_available'] = False
Path(sys.argv[1]).write_text(json.dumps(resources))
sys.exit(result)
