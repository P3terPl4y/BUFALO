#!/usr/bin/env python3
"""Exporta el entorno local antes de arrancar BUFALO o ejecutar artisan."""

import os
import shlex
from pathlib import Path
import sys


root = Path(__file__).resolve().parent.parent
os.chdir(root)
for raw in (root / ".env.production").read_text().splitlines():
    line = raw.strip()
    if not line or line.startswith("#") or "=" not in line:
        continue
    key, value = line.split("=", 1)
    key, value = key.strip(), value.strip()
    if value.startswith(("'", '"')):
        parsed = shlex.split(value, comments=False, posix=True)
        if len(parsed) != 1:
            raise ValueError("Invalid environment value")
        value = parsed[0]
    os.environ.setdefault(key, value)

os.environ["APP_ENV"] = "production"
os.environ["APP_DEBUG"] = "false"
os.environ.setdefault("APP_HOST", "127.0.0.1")
os.environ.setdefault("APP_PORT", "3000")
os.environ.setdefault("CSRF_TRUSTED_ORIGINS", "https://bufalo.duohnson.com")
binary = os.environ.get("BUFALO_BINARY", str(root / "bin" / "bufalo"))
os.execv(binary, [binary, *sys.argv[1:]])
