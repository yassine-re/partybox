import { createServer } from "node:http";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import { handler } from "./build/handler.js";

const host = process.env.HOST || "0.0.0.0";
const port = Number(process.env.PORT || 3000);
const apiTarget = new URL(
  process.env.API_INTERNAL_URL || "http://localhost:8080",
);
const websocketPath = /^\/api\/games\/[^/]+\/ws(?:\?|$)/;

const server = createServer(handler);

// SvelteKit request handlers cannot retain an upgraded connection. Relay only
// the realtime endpoint; ordinary /api requests keep using the existing route.
server.on("upgrade", (request, socket, head) => {
  if (!websocketPath.test(request.url || "")) {
    socket.end("HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n");
    return;
  }

  const send = apiTarget.protocol === "https:" ? httpsRequest : httpRequest;
  const upstreamRequest = send({
    protocol: apiTarget.protocol,
    hostname: apiTarget.hostname,
    port: apiTarget.port,
    method: request.method,
    path: request.url,
    headers: request.headers,
  });

  upstreamRequest.on("upgrade", (response, upstream, upstreamHead) => {
    const headers = [];
    for (let index = 0; index < response.rawHeaders.length; index += 2) {
      headers.push(`${response.rawHeaders[index]}: ${response.rawHeaders[index + 1]}`);
    }
    socket.write(
      `HTTP/1.1 ${response.statusCode} ${response.statusMessage}\r\n${headers.join("\r\n")}\r\n\r\n`,
    );
    if (head.length) upstream.write(head);
    if (upstreamHead.length) socket.write(upstreamHead);
    upstream.pipe(socket).pipe(upstream);
  });

  upstreamRequest.on("response", (response) => {
    socket.write(
      `HTTP/1.1 ${response.statusCode} ${response.statusMessage}\r\nConnection: close\r\n\r\n`,
    );
    response.pipe(socket);
  });
  upstreamRequest.on("error", () => {
    socket.end("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\n\r\n");
  });
  socket.on("error", () => upstreamRequest.destroy());
  upstreamRequest.end();
});

server.listen(port, host, () => {
  console.log(`PartyBox frontend listening on http://${host}:${port}`);
});
