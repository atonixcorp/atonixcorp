import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";

export async function GET(request: Request): Promise<NextResponse> {
  const apiURL = process.env.API_URL;
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" }, { status: 401 });
  }
  if (!apiURL) {
    return NextResponse.json({ error: "Compute API is unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
  const tenantID = new URL(request.url).searchParams.get("tenant_id");
  if (tenantID && !/^[1-9]\d*$/.test(tenantID)) {
    return NextResponse.json({ error: "Invalid workspace.", code: "INVALID_SCOPE" }, { status: 400 });
  }
  const path = tenantID
    ? `/api/v1/tenants/${encodeURIComponent(tenantID)}/projects`
    : "/api/v1/tenants";
  try {
    const response = await fetch(`${apiURL}${path}`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
      signal: AbortSignal.timeout(15000),
    });
    return new NextResponse(await response.arrayBuffer(), {
      status: response.status,
      headers: {
        "Content-Type": response.headers.get("content-type") || "application/json",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return NextResponse.json({ error: "Compute API is temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
}

export async function POST(request: Request): Promise<NextResponse> {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected." }, { status: 403 });
  }
  const apiURL = process.env.API_URL;
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required." }, { status: 401 });
  }
  if (!apiURL) {
    return NextResponse.json({ error: "Compute API is unavailable." }, { status: 503 });
  }

  let payload: { name?: unknown; slug?: unknown };
  try {
    payload = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid workspace request." }, { status: 400 });
  }
  if (typeof payload.name !== "string" || typeof payload.slug !== "string") {
    return NextResponse.json({ error: "Workspace name and slug are required." }, { status: 400 });
  }

  try {
    const response = await fetch(`${apiURL}/api/v1/tenants`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
      cache: "no-store",
      signal: AbortSignal.timeout(15000),
    });
    return new NextResponse(await response.arrayBuffer(), {
      status: response.status,
      headers: {
        "Content-Type": response.headers.get("content-type") || "application/json",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return NextResponse.json({ error: "Compute API is temporarily unavailable." }, { status: 503 });
  }
}
