FROM golang:1.26-alpine AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/provision-device ./cmd/provision-device

FROM alpine:3.23 AS runtime
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 partybox
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/provision-device /app/provision-device
COPY database/ /app/database/
ENV MIGRATIONS_DIR=/app/database/migrations SEED_FILE=/app/database/seed.sql GIN_MODE=release
USER partybox
EXPOSE 8080
ENTRYPOINT ["/app/api"]
