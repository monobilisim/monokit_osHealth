# Containerfile for running this plugin's tests under Podman.
#
# osHealth's tests drive a live systemd (TestCheckSystemInit calls systemctl),
# so the container boots systemd as PID 1 and MUST be run with `--systemd=always`
# — that is why we use Podman, not Docker (systemd-as-PID1 doesn't work on Docker).
# Invoke it via `just test` (or `just test TestName`), never by hand.
FROM golang:1.25-trixie

ENV container=podman
ENV DEBIAN_FRONTEND=noninteractive
ENV CGO_ENABLED=1

WORKDIR /app

# build-essential + CGO_ENABLED: lib uses the sqlite (cgo) driver.
# systemd / systemd-sysv / dbus: the systemd tests talk to a live systemctl.
# osHealth stores state in sqlite, so no database server is required.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential \
        e2fsprogs \
        systemd \
        systemd-sysv \
        dbus && \
    rm -rf /var/lib/apt/lists/*

STOPSIGNAL SIGRTMIN+3

# Copy the whole plugin (vlib is resolved locally via a go.mod replace; monokit_lib
# is fetched as a normal module) and warm the module cache.
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# lib.InitConfig reads config from /etc/mono.
RUN mkdir -p /etc/mono && \
    cp config/* /etc/mono/ && \
    chmod +x scripts/collect-test-artifacts.sh

# Test harness: tests.service runs `go test`, records the exit code, collects
# artifacts and powers off; exit-code.service propagates the code as the
# container's exit status.
RUN cp scripts/tests.service scripts/exit-code.service /etc/systemd/system/ && \
    cp scripts/exit.target /etc/systemd/system/ && \
    systemctl enable tests.service exit-code.service

ENTRYPOINT ["/sbin/init"]
