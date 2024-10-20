# syntax=docker/dockerfile:1

# ------------------------------------------------------------------------------

FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# download dependencies first
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /ws-chat .

# ------------------------------------------------------------------------------

FROM scratch

COPY --from=builder /ws-chat /ws-chat

ENTRYPOINT ["/ws-chat"]
