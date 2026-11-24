FROM golang:1.27.4-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY gen ./gen
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/blog ./cmd/server

FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/blog /app/blog
COPY api/openapi/blog.swagger.json /app/api/openapi/blog.swagger.json

USER 65532:65532

EXPOSE 8000 8080

ENTRYPOINT ["/app/blog"]
