ARG BUILD_FROM
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o whatsapp-bridge .

FROM $BUILD_FROM
RUN apk add --no-cache sqlite-libs ca-certificates

WORKDIR /app
COPY --from=builder /build/whatsapp-bridge .
COPY run.sh /
RUN chmod a+x /run.sh

CMD ["/run.sh"]
