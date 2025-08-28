FROM golang:1.25.0-bookworm AS builder

WORKDIR /go/src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY cmd/api/ ./cmd/api
COPY migrations/ ./migrations
RUN go build -ldflags="-w -s" -o /go/bin/api ./cmd/api

FROM gcr.io/distroless/base-debian12:latest

COPY --from=builder /go/bin/api /
USER nonroot:nonroot

ENTRYPOINT [ "/api" ]
