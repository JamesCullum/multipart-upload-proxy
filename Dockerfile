FROM golang:1.22-alpine AS build

WORKDIR /app

# Install VIPS
RUN apk add --update --no-cache build-base vips-heif vips-dev

# Compile Go
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /proxy .

## Deploy
FROM alpine:3.20

WORKDIR /

EXPOSE 6743

COPY --from=build /proxy /proxy

RUN apk add --update --no-cache vips-heif vips-dev && \
    addgroup nonroot && adduser --shell /sbin/nologin --disabled-password  --no-create-home --ingroup nonroot nonroot
USER nonroot:nonroot

ENTRYPOINT ["/proxy"]
