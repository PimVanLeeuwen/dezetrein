# dezetrein API image. Build from the backend folder:
#   docker build --build-arg VERSION=0.1.0 -t dezetrein-backend .

FROM golang:1.27.3-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/server ./cmd/server

# scratch: no shell or packages, only the binary and the CA bundle for HTTPS to NS.
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/server /server
USER 65532:65532
ENV LISTEN_ADDR=:8080
EXPOSE 8080
# Liveness: the binary asks itself for /healthz (no shell or curl in scratch). A stale NS feed is
# for the monitor to report, not a reason to mark the container unhealthy.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/server", "-healthcheck"]
ENTRYPOINT ["/server"]
