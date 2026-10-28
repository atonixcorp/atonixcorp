import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

const computeAPI = process.env.COMPUTE_API_URL;
const catalogPaths = {
  compute_types: "/api/v1/compute/types",
  regions: "/api/v1/compute/regions",
  images: "/api/v1/compute/images",
  plans: "/api/v1/compute/plans",
  networks: "/api/v1/compute/networks",
} as const;

async function fetchCatalog(path: string): Promise<unknown> {
  const response = await fetch(`${computeAPI}${path}`, {
    cache: "no-store",
    signal: AbortSignal.timeout(5000),
  });
  if (!response.ok) {
    throw new Error(`Compute catalog returned ${response.status}.`);
  }
  return response.json();
}

export async function GET(): Promise<NextResponse> {
  if (!computeAPI) {
    return NextResponse.json(
      { error: "Compute API is not configured.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503 },
    );
  }

  try {
    const [computeTypes, regions, images, plans, networks] = await Promise.all(
      Object.entries(catalogPaths).map(([, path]) => fetchCatalog(path)),
    );
    return NextResponse.json({
      compute_types: computeTypes,
      regions,
      images,
      plans,
      networks,
    }, {
      headers: { "Cache-Control": "no-store" },
    });
  } catch {
    return NextResponse.json(
      { error: "Compute catalog is temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503 },
    );
  }
}
