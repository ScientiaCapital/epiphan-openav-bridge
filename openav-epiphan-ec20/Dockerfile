FROM golang:1.25 AS builder

COPY source /go/src
COPY go.mod go.sum /go/

ENV GOPATH=

WORKDIR /go/src/microservice-framework
RUN rm -f go.mod go.sum
RUN go mod init github.com/Dartmouth-OpenAV/microservice-framework
RUN go mod tidy

WORKDIR /go
# The copied go.mod's replace path matches the host layout (./source/...); fix it up
# for the container's layout (./src/...).
RUN go mod edit -replace github.com/Dartmouth-OpenAV/microservice-framework=./src/microservice-framework
RUN go mod tidy

WORKDIR /go/src
# -tags timetzdata embeds the IANA tz database in the binary — Alpine's final stage has no
# /usr/share/zoneinfo, and the framework's logging panics on time.In() without it.
RUN CGO_ENABLED=0 go build -tags timetzdata -o /go/bin/microservice
# The framework binds :80 (framework.DefaultSocketPort) as a non-root user later in this
# build; grant just enough capability to bind a privileged port rather than running as root.
RUN apt-get update && apt-get install -y --no-install-recommends libcap2-bin \
    && setcap 'cap_net_bind_service=+ep' /go/bin/microservice

FROM alpine:3.20
RUN adduser -D -H -u 10001 svc
COPY --from=builder /go/bin/microservice /usr/local/bin/microservice
EXPOSE 80
USER svc
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:80/health || exit 1
ENTRYPOINT ["/usr/local/bin/microservice"]
