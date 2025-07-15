FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pkg pkg/
COPY cmd cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o arcgis-utils ./cmd/arcgis-utils

# Create wrapper script in builder stage
RUN echo '#!/bin/sh' > /app/arcgis-utils-wrapper && \
    echo 'exec /usr/local/bin/arcgis-utils -output="/results" "$@"' >> /app/arcgis-utils-wrapper && \
    chmod +x /app/arcgis-utils-wrapper

FROM chainguard/wolfi-base:latest

WORKDIR /results

COPY --from=builder /app/arcgis-utils /usr/local/bin/arcgis-utils
COPY --from=builder /app/arcgis-utils-wrapper /usr/local/bin/arcgis-utils-wrapper

LABEL org.opencontainers.image.title="ArcGIS Utils"
LABEL org.opencontainers.image.description="A Go-based tool for downloading data from various public ArcGIS sources and converting it to common geospatial formats"
LABEL org.opencontainers.image.source="https://github.com/Sudo-Ivan/arcgis-utils-go"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.authors="Sudo-Ivan"

ENTRYPOINT ["/usr/local/bin/arcgis-utils-wrapper"]
