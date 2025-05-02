FROM golang:1.24.2-bullseye as builder

WORKDIR /workspace
COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o service main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/service .

USER 1001:1001

ENTRYPOINT ["/service"]
