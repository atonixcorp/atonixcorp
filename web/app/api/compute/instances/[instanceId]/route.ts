import { NextResponse } from "next/server";

export async function GET(): Promise<NextResponse> {
  return NextResponse.json(
    {
      error: "Instance access requires authenticated user and tenant authorization.",
      code: "AUTHENTICATION_REQUIRED",
    },
    { status: 401, headers: { "Cache-Control": "no-store" } },
  );
}
