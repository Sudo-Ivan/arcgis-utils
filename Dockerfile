FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY pkg pkg
COPY cmd cmd

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o arcgis-utils ./cmd/arcgis-utils

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/arcgis-utils /arcgis-utils

LABEL org.opencontainers.image.title="ArcGIS Utils"
LABEL org.opencontainers.image.description="A Go-based tool for downloading data from various public ArcGIS sources and converting it to common geospatial formats"
LABEL org.opencontainers.image.source="https://github.com/Sudo-Ivan/arcgis-utils-go"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.authors="Sudo-Ivan"

ENTRYPOINT ["/arcgis-utils"] 