import { env } from "$env/dynamic/private";
import type { RequestHandler } from "./$types";

// Same-origin gateway: browsers never need to resolve the Docker hostname.
const proxy: RequestHandler = async ({ request, params, url }) => {
  const target = `${(env.API_INTERNAL_URL || "http://localhost:8080").replace(/\/$/, "")}/api/${params.path}${url.search}`;
  const headers = new Headers();
  for (const name of ["authorization", "content-type"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  const hasBody = request.method !== "GET" && request.method !== "HEAD";
  const body = hasBody ? await request.text() : undefined;
  if (body && new TextEncoder().encode(body).length > 8192) {
    return Response.json(
      { error: { message: "Requête trop volumineuse." } },
      { status: 413 },
    );
  }
  try {
    const response = await fetch(target, {
      method: request.method,
      headers,
      body,
      signal: AbortSignal.timeout(11_000),
      redirect: "error",
    });
    return new Response(response.body, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return Response.json(
      { error: { message: "La PartyBox est momentanément indisponible." } },
      { status: 502 },
    );
  }
};
export const GET = proxy;
export const POST = proxy;
