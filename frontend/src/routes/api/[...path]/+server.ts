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
  const isProof = request.method === "POST" && params.path === "players/me/mission/proof";
  const bodyLimit = isProof ? 5 * 1024 * 1024 + 64 * 1024 : 8192;
  if (Number(request.headers.get("content-length")) > bodyLimit) {
    return Response.json({ error: { message: "Requête trop volumineuse." } }, { status: 413 });
  }
  let body: Uint8Array<ArrayBuffer> | undefined;
  // Preserve multipart bytes and bound memory even without Content-Length.
  if (hasBody && request.body) {
    const reader = request.body.getReader();
    const chunks: Uint8Array[] = [];
    let size = 0;
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        size += value.byteLength;
        if (size > bodyLimit) {
          await reader.cancel();
          return Response.json({ error: { message: "Requête trop volumineuse." } }, { status: 413 });
        }
        chunks.push(value);
      }
      body = new Uint8Array(size);
      let offset = 0;
      for (const chunk of chunks) { body.set(chunk, offset); offset += chunk.byteLength; }
    } catch (error) {
      if (error && typeof error === "object" && "status" in error && error.status === 413) {
        return Response.json({ error: { message: "Requête trop volumineuse." } }, { status: 413 });
      }
      return Response.json({ error: { message: "Envoi de la photo interrompu." } }, { status: 400 });
    } finally {
      reader.releaseLock();
    }
  }
  try {
    const response = await fetch(target, {
      method: request.method,
      headers,
      body,
      signal: AbortSignal.timeout(isProof ? 65_000 : params.path?.endsWith("/ai-missions/generate") ? 50_000 : 11_000),
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
export const PUT = proxy;
