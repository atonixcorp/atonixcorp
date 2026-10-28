import { NextResponse } from "next/server";

export async function POST(request: Request): Promise<NextResponse> {
  const apiURL = process.env.API_URL;
  if (!apiURL) {
    return NextResponse.json({ error: "Account service is unavailable." }, { status: 503 });
  }

  let payload: { fullName?: string; email?: string; password?: string };
  try {
    payload = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid account request." }, { status: 400 });
  }
  if (typeof payload.fullName !== "string" ||
      typeof payload.email !== "string" ||
      typeof payload.password !== "string") {
    return NextResponse.json({ error: "Name, email, and password are required." }, { status: 400 });
  }

  try {
    const response = await fetch(`${apiURL}/api/v1/auth/signup`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
    const result = await response.json().catch(() => null) as { error?: string; message?: string } | null;
    if (!response.ok) {
      return NextResponse.json(
        { error: result?.error || "Could not create account." },
        { status: response.status },
      );
    }
    return NextResponse.json({ message: result?.message || "Account created." }, {
      status: 201,
      headers: { "Cache-Control": "no-store" },
    });
  } catch {
    return NextResponse.json({ error: "Account service is temporarily unavailable." }, { status: 503 });
  }
}
