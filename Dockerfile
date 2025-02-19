ARG GOLANG_VERSION=1.23.4

FROM golang:${GOLANG_VERSION}-bookworm as builder

ARG VIPS_VERSION=8.16.0
ARG LIBRAW_VERSION=0.21.3
ARG IMAGE_MAGICK_VERSION=7.1.1-43
ARG IMAGE_MAGICK_ENABLED=0
ARG TARGETARCH

ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
ENV VIPS_VERSION=${VIPS_VERSION}
ENV LIBRAW_VERSION=${LIBRAW_VERSION}
ENV IMAGE_MAGICK_ENABLED=${IMAGE_MAGICK_ENABLED}
ENV IMAGE_MAGICK_VERSION=${IMAGE_MAGICK_VERSION}

# Installs libvips + required libraries
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
      ca-certificates \
      automake \
      autoconf \
      build-essential \
      curl \
      meson \
      ninja-build \
      pkg-config \
      gobject-introspection \
      gtk-doc-tools \
      libglib2.0-dev \
      libjpeg62-turbo-dev \
      libpng-dev \
      libwebp-dev \
      libtiff-dev \
      libexif-dev \
      libxml2-dev \
      libpoppler-glib-dev \
      swig \
      libpango1.0-dev \
      libmatio-dev \
      libopenslide-dev \
      libcfitsio-dev \
      libopenjp2-7-dev \
      liblcms2-dev \
      libgsf-1-dev \
      libfftw3-dev \
      liborc-0.4-dev \
      librsvg2-dev \
      libimagequant-dev \
      libaom-dev \
      libheif-dev \
      libspng-dev \
      libcgif-dev

# run the build scripts
COPY docker/scripts/* ./
RUN ./build-libraw.sh
RUN ./build-imagemagick.sh
RUN ./build-libvips.sh

WORKDIR ${GOPATH}/src/github.com/cshum/imagor

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN if [ "$TARGETARCH" = "amd64" ]; then go test ./...; fi
RUN go build -o ${GOPATH}/bin/imagor ./cmd/imagor/main.go

FROM debian:bookworm-slim
LABEL maintainer="adrian@cshum.com"

COPY --from=builder /usr/local/lib /usr/local/lib
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# Install runtime dependencies
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  procps libglib2.0-0 libjpeg62-turbo libpng16-16 libopenexr-3-1-30 \
  libwebp7 libwebpmux3 libwebpdemux2 libtiff6 libexif12 libxml2 libpoppler-glib8 \
  libpango1.0-0 libmatio11 libopenslide0 libopenjp2-7 libjemalloc2 \
  libgsf-1-114 libfftw3-bin liborc-0.4-0 librsvg2-2 libcfitsio10 libimagequant0 libaom3 libheif1 \
  libspng0 libcgif0 \
  libltdl7 && \
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
