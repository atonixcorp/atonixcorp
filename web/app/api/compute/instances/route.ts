import { NextResponse } from "next/server";

const authenticationRequired = {
  error: "Instance access requires authenticated user and tenant authorization.",
  code: "AUTHENTICATION_REQUIRED",
};

export async function GET(): Promise<NextResponse> {
  return NextResponse.json(authenticationRequired, {
    status: 401,
    headers: { "Cache-Control": "no-store" },
  });
}

export async function POST(): Promise<NextResponse> {
  return NextResponse.json(authenticationRequired, {
    status: 401,
    headers: { "Cache-Control": "no-store" },
  });
}
