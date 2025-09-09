# --- build stage ---
FROM golang:1.24.5-bullseye AS builder

ENV GOSUMDB=off
RUN git config --global credential.helper store
RUN --mount=type=secret,id=org_token \
    echo "https://gha:$(cat /run/secrets/org_token)@github.com" > /root/.git-credentials

WORKDIR /workspace

# cgo needs a C toolchain
RUN apt-get update && apt-get install -y --no-install-recommends build-essential ca-certificates && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN --mount=type=secret,id=org_token go mod download

COPY ./ ./

# cgo ON; go-sqlite3 compiles its bundled sqlite (no system libsqlite needed)
ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64
RUN go build -tags sqlite_omit_load_extension -ldflags="-w -s" -o service ./main.go

# --- runtime stage ---
# distroless/cc contains the glibc runtime needed by cgo binaries
FROM gcr.io/distroless/cc-debian12:nonroot
WORKDIR /
COPY --from=builder /workspace/service .
USER 1001:1001
ENTRYPOINT ["/service"]
