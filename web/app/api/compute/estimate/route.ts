import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function POST(request: Request): Promise<NextResponse> {
  const computeAPI = process.env.COMPUTE_API_URL;
  if (!computeAPI) {
    return NextResponse.json(
      { error: "Compute API is not configured.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503 },
    );
  }

  try {
    const body = await request.json();
    const response = await fetch(`${computeAPI}/api/v1/compute/estimates`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
    const result = await response.json().catch(() => null);
    if (!response.ok) {
      return NextResponse.json(
        { error: "The selected configuration could not be estimated.", code: "PRICING_UNAVAILABLE" },
        { status: response.status === 422 ? 422 : 503 },
      );
    }
    return NextResponse.json(result, { headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json(
      { error: "Pricing is temporarily unavailable.", code: "PRICING_UNAVAILABLE" },
      { status: 503 },
    );
  }
}
