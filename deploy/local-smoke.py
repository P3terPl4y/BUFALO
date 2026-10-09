#!/usr/bin/env python3
"""Exercise cookies, CSRF and rate limits against the isolated local server."""
import http.cookiejar
import json
import re
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

BASE = "http://127.0.0.1:33333"
# The server trusts its loopback proxy. Give each run an isolated benchmark
# client identity so a previous run's 15-minute account quota does not mask it.
RUN_ID = uuid.uuid4().hex
CLIENT_IP = f"198.18.{int(RUN_ID[:2], 16)}.{int(RUN_ID[2:4], 16)}"
EMAIL = f"smoke-{RUN_ID}@test.com"


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


jar = http.cookiejar.CookieJar()
client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar), NoRedirect())


def request(path, data=None):
    req = urllib.request.Request(BASE + path,
          data=None if data is None else urllib.parse.urlencode(data).encode(),
          headers={"CF-Connecting-IP": CLIENT_IP})
    try:
        response = client.open(req, timeout=10)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        return response.status, response.headers, response.read().decode()


deadline = time.monotonic() + 10
while True:
    try:
        if request("/healthz")[0] == 200:
            break
    except urllib.error.URLError:
        pass
    if time.monotonic() >= deadline:
        raise SystemExit("local server did not become healthy")
    time.sleep(.1)

status, headers, _ = request("/css/style.css")
assert status == 200 and not headers.get("Set-Cookie"), "static file creates session"
assert headers.get("X-Content-Type-Options") == "nosniff"
for path in ["/healthz", "/readyz", "/login", "/register", "/register/confirm"]:
    status, _, body = request(path)
    assert status == 200, (path, status)
for path in ["/home", "/admin/users", "/admin/health/data"]:
    status, headers, _ = request(path)
    assert status == 303 and headers.get("Location", "").startswith("/login"), (path, status)
assert request("/missing-smoke-check")[0] == 404
for path in ["/login", "/register", "/register/confirm"]:
    assert request(path, {"email": EMAIL, "password": "invalid"})[0] == 403

statuses = []
for _ in range(12):
    status, _, body = request("/login")
    token = re.search(r'name="_csrf"\s+value="([^"]+)"', body).group(1)
    status, _, body = request("/login", {"email": EMAIL, "password": "invalid", "_csrf": token})
    assert status in (200, 429), status
    if status == 200:
        assert "Credenciales incorrectas" in body
    statuses.append(status)
assert 429 in statuses, "login rate limit not enforced"
assert 200 in statuses, "pre-existing quota: rerun after the login rate window expires"
assert request("/healthz")[0] == 200
print(json.dumps({"smoke": "passed", "login_statuses": statuses}))
