# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH VERSION=dev COMMIT=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o /out/lxp-bridge-go .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/lxp-bridge-go /lxp-bridge-go
USER 65532:65532
WORKDIR /var/lib/lxp-bridge-go/
EXPOSE 8080
ENTRYPOINT ["/lxp-bridge-go"]
CMD ["-configFile=/var/lib/lxp-bridge-go/config.yaml"]
