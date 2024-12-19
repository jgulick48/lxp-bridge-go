ARG ARCH=

FROM ${ARCH}golang:1.23.4 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY ./ ./

RUN go install golang.org/x/tools/cmd/stringer@latest

RUN go generate ./...

RUN GOOS=linux CGO_ENABLED=0 go build

FROM ${ARCH}alpine:3.21.0

COPY --from=builder /app/lxp-bridge-go /bin/lxp-bridge-go
WORKDIR /var/lib/lxp-bridge-go/

CMD ["/bin/lxp-bridge-go","-configFile=/var/lib/lxp-bridge-go/config.json"]