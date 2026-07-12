FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /listhaul ./cmd/listhaul

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /listhaul /usr/local/bin/listhaul

USER nonroot:nonroot
WORKDIR /data

ENTRYPOINT ["/usr/local/bin/listhaul"]
CMD ["run", "-c", "/config/listhaul.toml"]
