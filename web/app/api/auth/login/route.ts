import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export async function POST(request: Request): Promise<NextResponse> {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected." }, { status: 403 });
  }
  const apiURL = process.env.API_URL;
  if (!apiURL) {
    return NextResponse.json({ error: "Authentication service is unavailable." }, { status: 503 });
  }

  let credentials: { email?: string; password?: string };
  try {
    credentials = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid sign-in request." }, { status: 400 });
  }
  if (typeof credentials.email !== "string" || typeof credentials.password !== "string") {
    return NextResponse.json({ error: "Email and password are required." }, { status: 400 });
  }

  try {
    const response = await fetch(`${apiURL}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(credentials),
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
    const payload = await response.json() as { token?: unknown; user?: unknown; error?: string };
    if (!response.ok || typeof payload.token !== "string") {
      return NextResponse.json(
        { error: payload.error || "Email or password is incorrect." },
        { status: response.status === 401 ? 401 : 503 },
      );
    }

    const result = NextResponse.json({ user: payload.user }, {
      headers: { "Cache-Control": "no-store" },
    });
    result.cookies.set("atonixcorp_session", payload.token, {
      httpOnly: true,
      secure: !(
        process.env.NODE_ENV === "development" &&
        process.env.AUTH_COOKIE_SECURE?.toLowerCase() === "false"
      ),
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 24,
    });
    return result;
  } catch {
    return NextResponse.json({ error: "Authentication service is temporarily unavailable." }, { status: 503 });
  }
}
