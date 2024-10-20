# syntax=docker/dockerfile:1

# ------------------------------------------------------------------------------

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS base

ENV CGO_ENABLED=0

WORKDIR /app

# download and cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# ------------------------------------------------------------------------------

FROM base AS build

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
	GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /ws-chat .

# ------------------------------------------------------------------------------

FROM base AS test

RUN --mount=type=cache,target=/root/.cache/go-build \
go test -v .

# ------------------------------------------------------------------------------

FROM scratch

COPY --from=build /ws-chat /ws-chat

ENTRYPOINT ["/ws-chat"]
