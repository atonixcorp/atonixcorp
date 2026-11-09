import { NextResponse, type NextRequest } from "next/server";

function redirectWithReturnTo(path: string, request: NextRequest): NextResponse {
  const destination = new URL(path, request.url);
  destination.searchParams.set("next", `${request.nextUrl.pathname}${request.nextUrl.search}`);
  return NextResponse.redirect(destination);
}

export async function proxy(request: NextRequest): Promise<NextResponse> {
  const token = request.cookies.get("atonixcorp_session")?.value;
  if (!token) return redirectWithReturnTo("/login", request);

  const apiURL = process.env.API_URL?.replace(/\/+$/, "");
  if (!apiURL) return redirectWithReturnTo("/auth-unavailable", request);

  try {
    const response = await fetch(`${apiURL}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
    if (response.status === 401) {
      const redirect = redirectWithReturnTo("/login", request);
      redirect.cookies.delete("atonixcorp_session");
      return redirect;
    }
    if (response.status === 403) return NextResponse.redirect(new URL("/forbidden", request.url));
    if (!response.ok) return redirectWithReturnTo("/auth-unavailable", request);
    return NextResponse.next();
  } catch {
    return redirectWithReturnTo("/auth-unavailable", request);
  }
}

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/compute/:path*",
    "/containers/:path*",
    "/storage/:path*",
    "/billing/:path*",
    "/security/:path*",
    "/cloud/:path*",
    "/databases/:path*",
    "/networking/:path*",
    "/observability/:path*",
  ],
};