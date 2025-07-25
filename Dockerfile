ARG GOLANG_VERSION=1.24.5
FROM golang:${GOLANG_VERSION}-bookworm as builder

ARG VIPS_VERSION=8.17.1
ARG TARGETARCH
ARG ENABLE_MAGICK=false

ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig

# Installs libvips + required libraries + conditionally ImageMagick
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/backports.list && \
  DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  ca-certificates \
  automake build-essential curl \
  meson ninja-build pkg-config \
  gobject-introspection gtk-doc-tools libglib2.0-dev libjpeg62-turbo-dev libpng-dev \
  libwebp-dev libtiff-dev libexif-dev libxml2-dev libpoppler-glib-dev \
  swig libpango1.0-dev libmatio-dev libopenslide-dev libcfitsio-dev libopenjp2-7-dev liblcms2-dev \
  libgsf-1-dev libfftw3-dev liborc-0.4-dev librsvg2-dev libimagequant-dev libaom-dev \
  libspng-dev libcgif-dev && \
  apt-get install --no-install-recommends -y -t bookworm-backports libheif-dev libheif-plugin-x265 libheif-plugin-aomenc libjxl-dev && \
  # Conditionally install ImageMagick
  if [ "$ENABLE_MAGICK" = "true" ]; then \
    apt-get install --no-install-recommends -y libmagickwand-dev; \
  fi && \
  cd /tmp && \
    curl -fsSLO https://github.com/libvips/libvips/releases/download/v${VIPS_VERSION}/vips-${VIPS_VERSION}.tar.xz && \
    tar xf vips-${VIPS_VERSION}.tar.xz && \
    cd vips-${VIPS_VERSION} && \
    meson setup _build \
    --buildtype=release \
    --strip \
    --prefix=/usr/local \
    --libdir=lib \
    -Dmagick=$(if [ "$ENABLE_MAGICK" = "true" ]; then echo "enabled"; else echo "disabled"; fi) \
    -Djpeg-xl=enabled \
    -Dintrospection=disabled && \
    ninja -C _build && \
    ninja -C _build install && \
  ldconfig && \
  rm -rf /usr/local/lib/libvips-cpp.* && \
  rm -rf /usr/local/lib/*.a && \
  rm -rf /usr/local/lib/*.la

WORKDIR ${GOPATH}/src/github.com/cshum/imagor

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN if [ "$TARGETARCH" = "amd64" ]; then go test ./...; fi
RUN go build -o ${GOPATH}/bin/imagor ./cmd/imagor/main.go

FROM debian:bookworm-slim as runtime
LABEL maintainer="adrian@cshum.com"

ARG ENABLE_MAGICK=false

COPY --from=builder /usr/local/lib /usr/local/lib
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# Install runtime dependencies including conditionally ImageMagick
RUN echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/backports.list && \
  DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  procps curl libglib2.0-0 libjpeg62-turbo libpng16-16 libopenexr-3-1-30 \
  libwebp7 libwebpmux3 libwebpdemux2 libtiff6 libexif12 libxml2 libpoppler-glib8 \
  libpango1.0-0 libmatio11 libopenslide0 libopenjp2-7 libjemalloc2 \
  libgsf-1-114 libfftw3-bin liborc-0.4-0 librsvg2-2 libcfitsio10 libimagequant0 libaom3 \
  libspng0 libcgif0 && \
  apt-get install --no-install-recommends -y -t bookworm-backports libheif1 libheif-plugin-x265 libheif-plugin-aomenc libjxl0.7 && \
  # Conditionally install ImageMagick runtime
  if [ "$ENABLE_MAGICK" = "true" ]; then \
    apt-get install --no-install-recommends -y libmagickwand-6.q16-6; \
  fi && \
  ln -s /usr/lib/$(uname -m)-linux-gnu/libjemalloc.so.2 /usr/local/lib/libjemalloc.so && \
  apt-get autoremove -y && \
  apt-get autoclean && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=builder /go/bin/imagor /usr/local/bin/imagor

ENV VIPS_WARNING=0
ENV MALLOC_ARENA_MAX=2
ENV LD_PRELOAD=/usr/local/lib/libjemalloc.so

ENV PORT 8000

# use unprivileged user
USER nobody

ENTRYPOINT ["/usr/local/bin/imagor"]

EXPOSE ${PORT}
