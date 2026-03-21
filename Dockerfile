FROM golang:1.24 as build

ARG GOARCH=amd64

WORKDIR /go/src/app

# do this to benefit from caching
COPY go.mod .
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=${GOARCH} go build -ldflags="-w -s"
RUN chmod +x /go/src/app/lambdahttpgw

FROM debian:11-slim

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd gateway && useradd -rm -d /opt/gateway -s /bin/bash -g gateway gateway
USER gateway
WORKDIR /opt/gateway

COPY --from=build /go/src/app/lambdahttpgw /usr/local/bin/lambdahttpgw
ENTRYPOINT [ "lambdahttpgw" ]
