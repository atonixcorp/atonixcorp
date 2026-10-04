"""HTTP authentication and request-size policy for the agent API."""

import hmac
import ipaddress
from collections.abc import Mapping


class ApiError(Exception):
    """An expected HTTP API error."""

    def __init__(self, status: int, message: str):
        super().__init__(message)
        self.status = status
        self.message = message


class BearerTokenAuth:
    """Optional bearer-token authentication with public health-check access."""

    PUBLIC_PATHS = frozenset({"/healthz"})

    def __init__(self, token: str | None = None):
        if token is not None and (not isinstance(token, str) or not token.strip()):
            raise ValueError("API token must be a non-empty string when configured")
        self.token = token.strip() if token is not None else None

    def authorize(
        self,
        path: str,
        headers: Mapping[str, str],
    ) -> None:
        if path in self.PUBLIC_PATHS or self.token is None:
            return
        authorization = headers.get("Authorization", "")
        scheme, separator, supplied = authorization.partition(" ")
        if (
            not separator
            or scheme.casefold() != "bearer"
            or not hmac.compare_digest(supplied.strip(), self.token)
        ):
            raise ApiError(401, "A valid bearer token is required")


def require_safe_bind(host: str, token: str | None) -> None:
    """Require an API token before binding to any non-loopback address."""
    if host.casefold() == "localhost":
        return
    try:
        address = ipaddress.ip_address(host)
    except ValueError as error:
        raise ValueError(
            "Bind host must be localhost or an IP address; use a bearer token "
            "for non-loopback addresses"
        ) from error
    if not address.is_loopback and (token is None or not token.strip()):
        raise ValueError("AGENT_API_TOKEN is required to bind a non-loopback address")
