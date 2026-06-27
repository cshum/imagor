ARG GOLANG_VERSION=1.26.3
ARG BASE_IMAGE=ghcr.io/cshum/imagor-base:vips8.18.3-r13
ARG DEV_BASE_IMAGE=${BASE_IMAGE}-dev

FROM golang:${GOLANG_VERSION}-bookworm AS golang-base

FROM ${BASE_IMAGE} AS native-base

FROM ${DEV_BASE_IMAGE} AS builder

ARG GOLANG_VERSION
ARG ENABLE_MAGICK=false

COPY --from=golang-base /usr/local/go /usr/local/go

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:/go/bin:$PATH
ENV CGO_ENABLED=1
ENV PKG_CONFIG_PATH=/opt/imagor/lib/pkgconfig
ENV CGO_CFLAGS=-I/opt/imagor/include
ENV CGO_LDFLAGS="-L/opt/imagor/lib -Wl,-rpath,/opt/imagor/lib"

WORKDIR ${GOPATH}/src/github.com/cshum/imagor

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_CFLAGS_ALLOW=-Xpreprocessor go build -ldflags "-s -w" -o /opt/imagor/bin/imagor ./cmd/imagor/main.go

FROM native-base AS runtime
LABEL maintainer="adrian@cshum.com"

ARG ENABLE_MAGICK=false

RUN apt-get update \
  && apt-get upgrade -y \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    curl \
    media-types \
  && if [ "$ENABLE_MAGICK" = "true" ]; then \
    apt-get install -y --no-install-recommends imagemagick; \
  fi \
  && ln -s /usr/lib/$(uname -m)-linux-gnu/libjemalloc.so.2 /usr/local/lib/libjemalloc.so \
  && mkdir -p /var/cache/fontconfig \
  && chmod 777 /var/cache/fontconfig \
  && rm -rf /var/lib/apt/lists/* \
  && rm -rf /etc/fonts/conf.d/10-sub-pixel-rgb.conf /etc/fonts/conf.d/11-lcdfilter-default.conf

COPY --from=builder /opt/imagor/bin/imagor /opt/imagor/bin/imagor
RUN ln -s /opt/imagor/bin/imagor /usr/local/bin/imagor

ENV VIPS_WARNING=0
ENV MALLOC_ARENA_MAX=2
ENV LD_PRELOAD=/usr/local/lib/libjemalloc.so
ENV LD_LIBRARY_PATH=/opt/imagor/lib
ENV FONTCONFIG_PATH=/etc/fonts
ENV XDG_CACHE_HOME=/tmp
ENV VIPS_VECTOR=167772160

ENV PORT=8000

USER nobody

ENTRYPOINT ["/usr/local/bin/imagor"]

EXPOSE ${PORT}
