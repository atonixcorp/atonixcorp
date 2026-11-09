#!/usr/bin/env python3
"""Verify first-admin bootstrap and platform-wide authorization on a disposable stack."""

import json
import os
import pty
import secrets
import select
import subprocess
import time
import urllib.error
import urllib.request
import uuid
from http.cookies import SimpleCookie


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, response, code, message, headers, new_url):
        return None


opener = urllib.request.build_opener(NoRedirect)
web_url = os.environ.get("ATONIX_WEB_URL", "http://127.0.0.1:3000").rstrip("/")
api_url = os.environ.get("ATONIX_API_URL", "http://127.0.0.1:8080").rstrip("/")


def request(base_url, path, method="GET", body=None, token=None, headers=None):
    request_headers = dict(headers or {})
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        request_headers["Content-Type"] = "application/json"
    if token:
        request_headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(
        f"{base_url}{path}", data=data, headers=request_headers, method=method
    )
    try:
        with opener.open(req, timeout=15) as response:
            return response.status, response.headers, response.read()
    except urllib.error.HTTPError as error:
        return error.code, error.headers, error.read()


def expect_status(label, actual, expected):
    if actual != expected:
        raise AssertionError(f"{label}: got HTTP {actual}, expected {expected}")


def bootstrap_admin(full_name, email, password):
    master, slave = pty.openpty()
    compose_command = ["sudo", "-n", "docker", "compose"]
    for compose_file in os.environ.get("ATONIX_COMPOSE_FILES", "").split(os.pathsep):
        if compose_file:
            compose_command.extend(["-f", compose_file])
    project = os.environ.get("ATONIX_COMPOSE_PROJECT")
    if project:
        compose_command.extend(["--project-name", project])
    compose_command.extend(["exec", "api", "/app/atonix-admin"])
    process = subprocess.Popen(
        compose_command,
        stdin=slave,
        stdout=slave,
        stderr=slave,
        close_fds=True,
    )
    os.close(slave)
    transcript = bytearray()
    prompts = [
        (b"Super Admin full name: ", full_name),
        (b"Super Admin email: ", email),
        (b"Super Admin password (12-72 bytes): ", password),
        (b"Confirm password: ", password),
    ]
    prompt_index = 0
    deadline = time.monotonic() + 60
    try:
        while process.poll() is None and time.monotonic() < deadline:
            ready, _, _ = select.select([master], [], [], 0.2)
            if not ready:
                continue
            try:
                chunk = os.read(master, 4096)
            except OSError:
                break
            if not chunk:
                break
            transcript.extend(chunk)
            if prompt_index < len(prompts) and prompts[prompt_index][0] in transcript:
                os.write(master, prompts[prompt_index][1].encode() + b"\n")
                prompt_index += 1
        if process.poll() is None:
            process.kill()
            process.wait()
            raise AssertionError("interactive bootstrap did not finish")
        while select.select([master], [], [], 0)[0]:
            try:
                chunk = os.read(master, 4096)
            except OSError:
                break
            if not chunk:
                break
            transcript.extend(chunk)
        if prompt_index != len(prompts) or process.returncode != 0:
            raise AssertionError("interactive bootstrap failed")
        if password.encode() in transcript:
            raise AssertionError("bootstrap echoed the password to its terminal")
    finally:
        os.close(master)
        if process.poll() is None:
            process.kill()
            process.wait()


admin_email = f"ci-admin-{uuid.uuid4().hex}@example.invalid"
admin_password = secrets.token_urlsafe(24)
bootstrap_admin("CI Platform Administrator", admin_email, admin_password)

ordinary_email = f"ci-user-{uuid.uuid4().hex}@example.invalid"
ordinary_password = secrets.token_urlsafe(24)
expect_status(
    "ordinary signup",
    request(
        api_url,
        "/api/v1/auth/signup",
        "POST",
        {"fullName": "CI Ordinary User", "email": ordinary_email, "password": ordinary_password},
    )[0],
    201,
)

status, _, body = request(
    api_url,
    "/api/v1/auth/login",
    "POST",
    {"email": ordinary_email, "password": ordinary_password},
)
expect_status("ordinary login", status, 200)
ordinary_login = json.loads(body)
ordinary_token = ordinary_login["token"]
ordinary_id = ordinary_login["user"]["id"]
if ordinary_login["user"]["platformRole"] != "user":
    raise AssertionError("new signup did not receive the restricted user platform role")

status, _, body = request(
    api_url,
    "/api/v1/auth/login",
    "POST",
    {"email": admin_email, "password": admin_password},
)
expect_status("Super Admin login", status, 200)
admin_login = json.loads(body)
admin_token = admin_login["token"]
admin_id = admin_login["user"]["id"]
if admin_login["user"]["platformRole"] != "super_admin":
    raise AssertionError("bootstrapped account did not receive the Super Admin role")

expect_status("anonymous admin endpoint", request(api_url, "/api/v1/admin/users")[0], 401)
expect_status(
    "ordinary admin endpoint",
    request(api_url, "/api/v1/admin/users", token=ordinary_token)[0],
    403,
)
status, _, body = request(api_url, "/api/v1/admin/users", token=admin_token)
expect_status("Super Admin user listing", status, 200)
listed_users = json.loads(body)["users"]
if not any(user["id"] == ordinary_id and user["platformRole"] == "user" for user in listed_users):
    raise AssertionError("admin listing omitted the ordinary user or its restricted role")

status, _, body = request(api_url, "/api/v1/tenants", token=admin_token)
expect_status("Super Admin tenant listing", status, 200)
admin_tenants = json.loads(body)
ordinary_tenants = json.loads(request(api_url, "/api/v1/tenants", token=ordinary_token)[2])
admin_tenant = next(tenant for tenant in admin_tenants if tenant["slug"] == f"tenant-{admin_id}")
ordinary_tenant = next(tenant for tenant in admin_tenants if tenant["slug"] == f"tenant-{ordinary_id}")
if [tenant["id"] for tenant in ordinary_tenants] != [ordinary_tenant["id"]]:
    raise AssertionError("ordinary user tenant listing escaped its membership scope")
expect_status(
    "ordinary cross-tenant project access",
    request(
        api_url,
        f"/api/v1/tenants/{admin_tenant['id']}/projects",
        token=ordinary_token,
    )[0],
    404,
)
expect_status(
    "Super Admin cross-tenant project access",
    request(
        api_url,
        f"/api/v1/tenants/{admin_tenant['id']}/projects",
        token=admin_token,
    )[0],
    200,
)
admin_projects = json.loads(
    request(
        api_url,
        f"/api/v1/tenants/{admin_tenant['id']}/projects",
        token=admin_token,
    )[2]
)
admin_project_id = admin_projects[0]["id"]
ordinary_cookie = {"Cookie": f"atonixcorp_session={ordinary_token}"}
admin_cookie = {"Cookie": f"atonixcorp_session={admin_token}"}
service_routes = (
    (
        f"/api/databases/tenants/{admin_tenant['id']}/projects/{admin_project_id}/databases",
        "database service",
    ),
    (
        f"/api/networking/tenants/{admin_tenant['id']}/projects/{admin_project_id}/networks",
        "network service",
    ),
)
for service_path, service_name in service_routes:
    expect_status(
        f"ordinary cross-tenant {service_name} access",
        request(web_url, service_path, headers=ordinary_cookie)[0],
        403 if service_name == "database service" else 404,
    )
    expect_status(
        f"Super Admin cross-tenant {service_name} access",
        request(web_url, service_path, headers=admin_cookie)[0],
        200,
    )

expect_status(
    "promote user",
    request(
        api_url,
        f"/api/v1/admin/users/{ordinary_id}/platform-role",
        "PATCH",
        {"platformRole": "super_admin"},
        token=admin_token,
    )[0],
    200,
)
expect_status(
    "promoted user's role takes effect immediately",
    request(api_url, "/api/v1/admin/users", token=ordinary_token)[0],
    200,
)
for service_path, service_name in service_routes:
    expect_status(
        f"promoted user's cross-tenant {service_name} access",
        request(web_url, service_path, headers=ordinary_cookie)[0],
        200,
    )
expect_status(
    "restore ordinary user role",
    request(
        api_url,
        f"/api/v1/admin/users/{ordinary_id}/platform-role",
        "PATCH",
        {"platformRole": "user"},
        token=admin_token,
    )[0],
    200,
)
expect_status(
    "demoted user's old token loses admin access",
    request(api_url, "/api/v1/admin/users", token=ordinary_token)[0],
    403,
)
for service_path, service_name in service_routes:
    expect_status(
        f"demoted user's cross-tenant {service_name} access",
        request(web_url, service_path, headers=ordinary_cookie)[0],
        403 if service_name == "database service" else 404,
    )
expect_status(
    "prevent last Super Admin demotion",
    request(
        api_url,
        f"/api/v1/admin/users/{admin_id}/platform-role",
        "PATCH",
        {"platformRole": "user"},
        token=admin_token,
    )[0],
    409,
)

origin = {"Origin": web_url}
status, headers, _ = request(
    web_url,
    "/api/auth/login",
    "POST",
    {"email": admin_email, "password": admin_password},
    headers=origin,
)
expect_status("web Super Admin login", status, 200)
cookie = SimpleCookie()
cookie.load(headers.get("Set-Cookie", ""))
if "atonixcorp_session" not in cookie:
    raise AssertionError("web login did not establish the Super Admin session")
cookie_header = {"Cookie": f"atonixcorp_session={cookie['atonixcorp_session'].value}"}
status, _, body = request(web_url, "/api/auth/session", headers=cookie_header)
expect_status("web Super Admin session", status, 200)
if json.loads(body)["user"]["platformRole"] != "super_admin":
    raise AssertionError("web session did not identify the Super Admin role")
status, _, _ = request(
    web_url, "/api/auth/logout", "POST", {}, headers={**origin, **cookie_header}
)
expect_status("web Super Admin logout", status, 200)

print("Super Admin smoke test passed: bootstrap, login, role enforcement, cross-tenant access, role management, and last-admin protection.")
