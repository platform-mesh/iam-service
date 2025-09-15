FROM golang:1.24.6-bullseye AS builder

ENV GOSUMDB=off
RUN git config --global credential.helper store
RUN --mount=type=secret,id=org_token \
    echo "https://gha:$(cat /run/secrets/org_token)@github.com" > /root/.git-credentials


WORKDIR /workspace

COPY go.mod go.sum ./
RUN --mount=type=secret,id=org_token go mod download

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o service main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/service .

USER 1001:1001

ENTRYPOINT ["/service"]
