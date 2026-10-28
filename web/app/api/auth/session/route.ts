import { cookies } from "next/headers";
import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function GET(): Promise<NextResponse> {
  const apiURL = process.env.API_URL;
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required." }, { status: 401 });
  }
  if (!apiURL) {
    return NextResponse.json({ error: "Authentication service is unavailable." }, { status: 503 });
  }

  try {
    const response = await fetch(`${apiURL}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
    const payload = await response.json().catch(() => null);
    if (!response.ok) {
      const result = NextResponse.json(
        { error: "Your session is no longer valid." },
        { status: response.status === 401 ? 401 : 503 },
      );
      if (response.status === 401) result.cookies.delete("atonixcorp_session");
      return result;
    }
    return NextResponse.json(payload, { headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "Authentication service is temporarily unavailable." }, { status: 503 });
  }
}
