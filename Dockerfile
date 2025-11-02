# syntax=docker/dockerfile:1

FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/blog-api ./cmd/api

FROM alpine:3.19
RUN adduser -D -g '' appuser
WORKDIR /app

COPY --from=builder /app/bin/blog-api /usr/local/bin/blog-api
COPY --from=builder /app/themes ./themes
COPY --from=builder /app/plugins ./plugins
COPY --from=builder /app/static ./static
COPY favicon.ico ./favicon.ico

RUN mkdir -p /app/uploads \
    && chown -R appuser:appuser /app /usr/local/bin/blog-api

USER appuser
EXPOSE 8080
ENV PORT=8080
VOLUME ["/app/uploads"]

ENTRYPOINT ["/usr/local/bin/blog-api"]
