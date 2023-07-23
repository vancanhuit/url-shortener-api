FROM golang:1.20-bullseye as builder

WORKDIR /go/src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY cmd/api/ ./cmd/api
COPY migrations/ ./migrations
RUN go build -o /go/bin/api ./cmd/api

FROM gcr.io/distroless/base-debian11

COPY --from=builder /go/bin/api /
USER nonroot:nonroot

ENTRYPOINT [ "/api" ]
