ARG GOLANG_VERSION=1.25.1
FROM golang:${GOLANG_VERSION}-trixie as builder

ARG VIPS_VERSION=8.17.2
ARG TARGETARCH
ARG ENABLE_MAGICK=false
ARG ENABLE_MOZJPEG=false

ARG MOZJPEG_VERSION=4.1.1
ARG MOZJPEG_URL=https://github.com/mozilla/mozjpeg/archive

ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig

# Conditionally install MozJPEG build dependencies and build MozJPEG
RUN if [ "$ENABLE_MOZJPEG" = "true" ]; then \
  DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y build-essential libboost-all-dev pkg-config autoconf automake libtool nasm make cmake flex libpng-tools libpng-dev zlib1g-dev && \
  cd /tmp && \
  curl -fsSLO ${MOZJPEG_URL}/v${MOZJPEG_VERSION}.tar.gz && \
  tar xf v${MOZJPEG_VERSION}.tar.gz && \
  cd mozjpeg-${MOZJPEG_VERSION} && \
  cmake -G"Unix Makefiles" . && \
  make -j4 && \
  make install && \
  cp jpegint.h /usr/include/jpegint.h && \
  cd .. && \
  rm -rf mozjpeg-${MOZJPEG_VERSION} v${MOZJPEG_VERSION}.tar.gz; \
fi

# Installs libvips + required libraries + conditionally ImageMagick
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  ca-certificates \
  automake build-essential curl \
  meson ninja-build pkg-config \
  gobject-introspection gtk-doc-tools libglib2.0-dev libjpeg62-turbo-dev libpng-dev \
  libwebp-dev libtiff-dev libexif-dev libxml2-dev libpoppler-glib-dev \
  swig libpango1.0-dev libmatio-dev libopenslide-dev libcfitsio-dev libopenjp2-7-dev liblcms2-dev \
  libgsf-1-dev libfftw3-dev liborc-0.4-dev librsvg2-dev libimagequant-dev libaom-dev \
  libspng-dev libcgif-dev libheif-dev libheif-plugin-x265 libheif-plugin-aomenc libjxl-dev libavif-dev && \
  if [ "$ENABLE_MAGICK" = "true" ]; then \
    apt-get install --no-install-recommends -y libmagickwand-dev; \
  fi && \
  cd /tmp && \
    if [ "$ENABLE_MOZJPEG" = "true" ]; then \
      export PKG_CONFIG_PATH=/opt/mozjpeg/lib64/pkgconfig/; \
    fi && \
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

RUN go build -o ${GOPATH}/bin/imagor ./cmd/imagor/main.go

FROM debian:trixie-slim as runtime
LABEL maintainer="adrian@cshum.com"

ARG ENABLE_MAGICK=false
ARG ENABLE_MOZJPEG=false

COPY --from=builder /usr/local/lib /usr/local/lib
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# Conditionally copy MozJPEG libraries
COPY --from=builder /opt/mozjpeg /opt/mozjpeg
RUN if [ "$ENABLE_MOZJPEG" != "true" ]; then \
  rm -rf /opt/mozjpeg; \
fi

# Install runtime dependencies including conditionally ImageMagick
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  procps curl libglib2.0-0 libjpeg62-turbo libpng16-16 libopenexr-3-1-30 \
  libwebp7 libwebpmux3 libwebpdemux2 libtiff6 libexif12 libxml2 libpoppler-glib8t64 \
  libpango-1.0-0 libmatio13 libopenslide0 libopenjp2-7 libjemalloc2 \
  libgsf-1-114 libfftw3-bin liborc-0.4-0 librsvg2-2 libcfitsio10t64 libimagequant0 libaom3 \
  libspng0 libcgif0 libheif1 libheif-plugin-x265 libheif-plugin-aomenc libjxl0.11 && \
  if [ "$ENABLE_MAGICK" = "true" ]; then \
    apt-get install --no-install-recommends -y libmagickwand-7.q16-10; \
  fi && \
  ln -s /usr/lib/$(uname -m)-linux-gnu/libjemalloc.so.2 /usr/local/lib/libjemalloc.so && \
  apt-get autoremove -y && \
  apt-get autoclean && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=builder /go/bin/imagor /usr/local/bin/imagor

# Set MozJPEG environment variables conditionally
ENV VIPS_WARNING=0
ENV MALLOC_ARENA_MAX=2
ENV LD_PRELOAD=/usr/local/lib/libjemalloc.so
ENV PKG_CONFIG_PATH=${ENABLE_MOZJPEG:+/opt/mozjpeg/lib64/pkgconfig/}
ENV LD_LIBRARY_PATH=${ENABLE_MOZJPEG:+/opt/mozjpeg/lib64}

ENV PORT 8000

# use unprivileged user
USER nobody

ENTRYPOINT ["/usr/local/bin/imagor"]

EXPOSE ${PORT}
