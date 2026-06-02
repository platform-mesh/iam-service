FROM --platform=$BUILDPLATFORM golang:1.26.3-trixie@sha256:0f6b034c99663ea8957e7dae99124e37374cbe7fcb5b5646f19b185f8f976279 AS builder
ARG TARGETARCH
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags '-w -s' -o service main.go

FROM gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240
WORKDIR /
COPY --from=builder /workspace/service .

USER 1001:1001

ENTRYPOINT ["/service"]
