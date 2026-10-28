import { NextResponse } from "next/server";

const apiUrl = process.env.API_URL ?? "http://localhost:8080";
const maxAttempts = 5;
const windowMs = 10 * 60 * 1000;
const rateLimitByIp = new Map<string, { attempts: number; expiresAt: number }>();

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isString(value: unknown, maxLength: number, required = false): value is string {
  return (
    typeof value === "string" &&
    value.length <= maxLength &&
    (!required || value.trim().length > 0)
  );
}

function clientIp(request: Request) {
  const forwardedFor = request.headers.get("x-forwarded-for");
  return (
    request.headers.get("x-real-ip")?.trim() ||
    forwardedFor?.split(",")[0]?.trim() ||
    "unknown"
  );
}

function isRateLimited(ip: string) {
  const now = Date.now();
  for (const [key, value] of rateLimitByIp) {
    if (value.expiresAt <= now) rateLimitByIp.delete(key);
  }

  const current = rateLimitByIp.get(ip);
  if (!current || current.expiresAt <= now) {
    if (rateLimitByIp.size >= 5000) {
      const oldestKey = rateLimitByIp.keys().next().value;
      if (oldestKey !== undefined) rateLimitByIp.delete(oldestKey);
    }
    rateLimitByIp.set(ip, { attempts: 1, expiresAt: now + windowMs });
    return false;
  }

  current.attempts += 1;
  return current.attempts > maxAttempts;
}

export async function POST(request: Request) {
  const contentLength = Number(request.headers.get("content-length") ?? 0);
  if (contentLength > 16_000) {
    return NextResponse.json({ error: "Request is too large." }, { status: 413 });
  }

  let payload: unknown;
  try {
    payload = await request.json();
  } catch {
    return NextResponse.json({ error: "Request body must be valid JSON." }, { status: 400 });
  }

  if (!isObject(payload)) {
    return NextResponse.json({ error: "Enter the required sponsorship details." }, { status: 400 });
  }

  if (isString(payload.website, 500) && payload.website.trim()) {
    return NextResponse.json({ error: "Sponsorship request rejected." }, { status: 400 });
  }

  if (isRateLimited(clientIp(request))) {
    return NextResponse.json(
      { error: "Too many requests. Please wait before submitting again." },
      { status: 429 },
    );
  }

  const amount = payload.amount;
  const validAmount =
    typeof amount === "number" && Number.isFinite(amount) && amount > 0 && amount <= 10_000_000;
  const validProvider =
    payload.paymentProvider === "payfast" ||
    payload.paymentProvider === "stripe" ||
    payload.paymentProvider === "paypal" ||
    payload.paymentProvider === "bank-transfer" ||
    payload.paymentProvider === "not-sure";
  const validPackage =
    payload.package === "" ||
    payload.package === "bronze" ||
    payload.package === "silver" ||
    payload.package === "gold" ||
    payload.package === "platinum";

  if (
    !isString(payload.name, 255, true) ||
    !isString(payload.company, 255) ||
    !isString(payload.email, 255, true) ||
    !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(payload.email.trim()) ||
    !isString(payload.phone, 50) ||
    !isString(payload.country, 100, true) ||
    !validAmount ||
    !validProvider ||
    !validPackage ||
    !isString(payload.message, 10_000)
  ) {
    return NextResponse.json(
      { error: "Check the required details and ensure each field is within the allowed length." },
      { status: 400 },
    );
  }

  let response: Response;
  try {
    response = await fetch(`${apiUrl}/api/v1/contact/sponsorship`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: payload.name.trim(),
        company: payload.company.trim(),
        email: payload.email.trim(),
        phone: payload.phone.trim(),
        country: payload.country.trim(),
        amount,
        package: payload.package,
        paymentProvider: payload.paymentProvider,
        message: payload.message.trim(),
      }),
      signal: AbortSignal.timeout(10_000),
    });
  } catch {
    return NextResponse.json({ error: "The sponsorship service is unavailable." }, { status: 502 });
  }

  if (!response.ok) {
    const responseText = await response.text();
    let errorMessage = "The sponsorship service could not process your request.";
    try {
      const serviceResponse: unknown = JSON.parse(responseText);
      if (
        isObject(serviceResponse) &&
        typeof serviceResponse.error === "string" &&
        serviceResponse.error.length <= 500
      ) {
        errorMessage = serviceResponse.error;
      }
    } catch {
      // Keep the public response generic when the upstream response is not JSON.
    }
    return NextResponse.json({ error: errorMessage }, { status: response.status });
  }

  return NextResponse.json({ message: "Sponsorship request received." }, { status: 200 });
}
