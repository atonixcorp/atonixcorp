import { NextResponse } from "next/server";
import { cookies } from "next/headers";

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
  const incoming = new URL(request.url);
  const tenantID = incoming.searchParams.get("tenant_id") ?? "";
  const projectID = incoming.searchParams.get("project_id") ?? "";
  if (!/^[1-9]\d*$/.test(tenantID) || !/^[1-9]\d*$/.test(projectID)) {
    return NextResponse.json({ error: "A workspace and project are required.", code: "INVALID_SCOPE" }, { status: 400 });
  }
  try {
    const response = await fetch(
      `${apiURL}/api/v1/cloud/catalog?tenant_id=${encodeURIComponent(tenantID)}&project_id=${encodeURIComponent(projectID)}`,
      {
        headers: { Authorization: `Bearer ${token}` },
        cache: "no-store",
        signal: AbortSignal.timeout(30000),
      },
    );
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
