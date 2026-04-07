FROM --platform=$BUILDPLATFORM golang:1.26.1-trixie@sha256:1d414b0376b53ec94b9a2493229adb81df8b90af014b18619732f1ceaaf7234a AS builder
ARG TARGETARCH
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags '-w -s' -o service main.go

FROM gcr.io/distroless/static:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39
WORKDIR /
COPY --from=builder /workspace/service .

USER 1001:1001

ENTRYPOINT ["/service"]
