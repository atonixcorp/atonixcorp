import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function POST(): Promise<NextResponse> {
  return NextResponse.json(
    {
      error: "Pricing is unavailable for this OpenStack account.",
      code: "PRICING_UNAVAILABLE",
      pricing_available: false,
    },
    { status: 503, headers: { "Cache-Control": "no-store" } },
  );
}
