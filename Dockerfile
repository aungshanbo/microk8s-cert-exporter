FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w" \
  -o microk8s-cert-exporter

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/microk8s-cert-exporter /microk8s-cert-exporter

EXPOSE 9101

ENTRYPOINT ["/microk8s-cert-exporter"]