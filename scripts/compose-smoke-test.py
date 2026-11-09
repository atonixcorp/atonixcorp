#!/usr/bin/env python3
"""Exercise the root Compose stack's public web and authentication flows."""

import json
import os
import secrets
import urllib.error
import urllib.request
import uuid
from datetime import datetime, timezone
from email.utils import parsedate_to_datetime
from http.cookies import SimpleCookie
from urllib.parse import urlsplit


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, response, code, message, headers, new_url):
        return None


opener = urllib.request.build_opener(NoRedirect)
web_url = os.environ.get("ATONIX_WEB_URL", "http://127.0.0.1:3000").rstrip("/")
api_url = os.environ.get("ATONIX_API_URL", "http://127.0.0.1:8080").rstrip("/")


def request(base_url, path, method="GET", body=None, headers=None):
    request_headers = dict(headers or {})
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        request_headers["Content-Type"] = "application/json"
    req = urllib.request.Request(
        f"{base_url}{path}", data=data, headers=request_headers, method=method
    )
    try:
        with opener.open(req, timeout=15) as response:
            return response.status, response.headers, response.read()
    except urllib.error.HTTPError as error:
        return error.code, error.headers, error.read()


def check_status(label, actual, expected):
    if actual != expected:
        raise AssertionError(f"{label}: got HTTP {actual}, expected {expected}")


check_status("API health", request(api_url, "/health")[0], 200)
check_status("Web login page", request(web_url, "/login")[0], 200)
check_status("Agent proxy", request(web_url, "/api/agent/registry")[0], 200)
check_status("Unauthenticated API session", request(api_url, "/api/v1/auth/me")[0], 401)

origin = {"Origin": web_url}
email = f"ci-{uuid.uuid4().hex}@example.invalid"
password = secrets.token_urlsafe(32)
account = {"fullName": "Compose Smoke Test", "email": email, "password": password}

check_status(
    "Cross-origin signup rejection",
    request(
        web_url,
        "/api/auth/signup",
        "POST",
        account,
        {"Origin": "https://attacker.invalid"},
    )[0],
    403,
)
check_status(
    "Signup",
    request(web_url, "/api/auth/signup", "POST", account, origin)[0],
    201,
)
check_status(
    "Invalid login",
    request(
        web_url,
        "/api/auth/login",
        "POST",
        {"email": email, "password": "invalid-password"},
        origin,
    )[0],
    401,
)

status, headers, body = request(
    web_url,
    "/api/auth/login",
    "POST",
    {"email": email, "password": password},
    origin,
)
check_status("Login", status, 200)
if json.loads(body)["user"]["email"] != email:
    raise AssertionError("Login returned the wrong user")

cookie = SimpleCookie()
cookie.load(headers.get("Set-Cookie", ""))
if "atonixcorp_session" not in cookie:
    raise AssertionError("Login did not set the session cookie")
session_cookie = cookie["atonixcorp_session"]
if "httponly" not in session_cookie or session_cookie["samesite"].lower() != "lax":
    raise AssertionError("Session cookie is missing HttpOnly or SameSite=Lax")
api_auth_headers = {"Authorization": f"Bearer {session_cookie.value}"}
tenants = json.loads(request(api_url, "/api/v1/tenants", headers=api_auth_headers)[2])
projects = json.loads(
    request(
        api_url,
        f"/api/v1/tenants/{tenants[0]['id']}/projects",
        headers=api_auth_headers,
    )[2]
)
check_status(
    "Storage volume types",
    request(
        api_url,
        f"/api/v1/storage/volume-types?tenant_id={tenants[0]['id']}&project_id={projects[0]['id']}",
        headers=api_auth_headers,
    )[0],
    200,
)
secure_expected = os.environ.get("AUTH_COOKIE_SECURE", "true").lower() != "false"
if ("secure" in session_cookie) != secure_expected:
    raise AssertionError("Session cookie Secure flag does not match configuration")

cookie_header = f"atonixcorp_session={session_cookie.value}"
auth_headers = {"Cookie": cookie_header}
check_status("Session lookup", request(web_url, "/api/auth/session", headers=auth_headers)[0], 200)
check_status("Authenticated dashboard", request(web_url, "/dashboard", headers=auth_headers)[0], 200)

status, headers, _ = request(
    web_url, "/api/auth/logout", "POST", {}, {**origin, **auth_headers}
)
check_status("Logout", status, 200)
cleared_cookie = SimpleCookie()
cleared_cookie.load(headers.get("Set-Cookie", ""))
session_cookie = cleared_cookie.get("atonixcorp_session")
expired = bool(session_cookie and session_cookie["max-age"] == "0")
if session_cookie and not expired and session_cookie["expires"]:
    try:
        expired = parsedate_to_datetime(session_cookie["expires"]) <= datetime.now(timezone.utc)
    except (TypeError, ValueError):
        expired = False
if not expired:
    raise AssertionError("Logout did not expire the session cookie")
check_status("Session after logout", request(web_url, "/api/auth/session")[0], 401)

status, headers, _ = request(web_url, "/dashboard")
location = urlsplit(headers.get("Location", ""))
if status not in (301, 302, 307, 308) or location.path != "/login":
    raise AssertionError("Unauthenticated dashboard access was not redirected to login")

print("Compose smoke test passed: services, agent proxy, storage, signup, login, session, logout, and route protection.")
