import { NextResponse } from "next/server";

export async function GET(): Promise<NextResponse> {
  return NextResponse.json(
    {
      error: "SSH key access requires an authenticated user and tenant session.",
      code: "AUTHENTICATION_REQUIRED",
    },
    { status: 401, headers: { "Cache-Control": "no-store" } },
  );
}
