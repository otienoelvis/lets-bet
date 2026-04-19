FROM golang:1.26-alpine AS builder

ARG SERVICE=gateway

WORKDIR /src

RUN apk add --no-cache git ca-certificates curl

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Download MaxMind GeoLite2 database
RUN mkdir -p /src/data && \
    curl -L "https://github.com/maxmind/GeoIP2/raw/main/GeoLite2-Country.mmdb.gz" | \
    gunzip -c > /src/data/GeoLite2-Country.mmdb

RUN CGO_ENABLED=0 GOOS=linux GOFLAGS="-trimpath" \
    go build -ldflags="-s -w" -o /out/app ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/app /app/app
COPY --from=builder /src/migrations /app/migrations
COPY --from=builder /src/data /app/data

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/app"]
