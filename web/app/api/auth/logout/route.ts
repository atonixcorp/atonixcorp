import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export async function POST(request: Request): Promise<NextResponse> {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected." }, { status: 403 });
  }
  const response = NextResponse.json({ success: true }, {
    headers: { "Cache-Control": "no-store" },
  });
  response.cookies.delete("atonixcorp_session");
  return response;
}
