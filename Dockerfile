# Cross-compile on the runner arch, then pack a static binary for the target.
FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X github.com/Fortumers/Addon-Mikan/internal/app.Version=${VERSION}" \
    -o /out/bot ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bot /bot
USER 65532:65532
ENTRYPOINT ["/bot"]
