import { NextResponse } from "next/server";

const apiUrl = process.env.API_URL ?? "http://localhost:8080";

export async function POST(request: Request) {
  let payload: unknown;
  try {
    payload = await request.json();
  } catch {
    return NextResponse.json({ error: "Request body must be valid JSON." }, { status: 400 });
  }

  if (
    !payload ||
    typeof payload !== "object" ||
    !("name" in payload) ||
    !("email" in payload) ||
    !("subject" in payload) ||
    !("message" in payload) ||
    typeof payload.name !== "string" ||
    typeof payload.email !== "string" ||
    typeof payload.subject !== "string" ||
    typeof payload.message !== "string" ||
    !payload.name.trim() ||
    !payload.email.trim() ||
    !payload.subject.trim() ||
    !payload.message.trim() ||
    payload.name.length > 255 ||
    payload.email.length > 255 ||
    payload.subject.length > 255 ||
    payload.message.length > 10000 ||
    !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(payload.email.trim())
  ) {
    return NextResponse.json(
      { error: "Enter a valid name, email, subject, and message within the allowed lengths." },
      { status: 400 },
    );
  }

  let response: Response;
  try {
    response = await fetch(`${apiUrl}/api/v1/contact/submit`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch {
    return NextResponse.json({ error: "The contact service is unavailable." }, { status: 502 });
  }

  const responseText = await response.text();
  let data: unknown;
  try {
    data = JSON.parse(responseText);
  } catch {
    return NextResponse.json({ error: "The contact service returned an invalid response." }, { status: 502 });
  }

  return NextResponse.json(data, { status: response.status });
}
