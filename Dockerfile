ARG GOLANG_VERSION=1.18.3
FROM golang:${GOLANG_VERSION}-bullseye as builder

ARG VIPS_VERSION=8.12.2
ARG CGIF_VERSION=0.3.0
ARG RUN_TEST

ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig

# libaom3 is in Debian bullseye-backports 
RUN echo 'deb http://deb.debian.org/debian bullseye-backports main' > /etc/apt/sources.list.d/backports.list

# Installs libvips + required libraries
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  ca-certificates \
  automake build-essential curl \
  python3-pip ninja-build pkg-config \
  gobject-introspection gtk-doc-tools libglib2.0-dev libjpeg62-turbo-dev libpng-dev \
  libwebp-dev libtiff5-dev libexif-dev libxml2-dev libpoppler-glib-dev \
  swig libpango1.0-dev libmatio-dev libopenslide-dev libcfitsio-dev \
  libgsf-1-dev fftw3-dev liborc-0.4-dev librsvg2-dev libimagequant-dev libaom-dev/bullseye-backports libheif-dev && \
  pip3 install meson && \
  cd /tmp && \
  curl -fsSLO https://github.com/dloebl/cgif/archive/refs/tags/V${CGIF_VERSION}.tar.gz && \
  tar xf V${CGIF_VERSION}.tar.gz && \
  cd cgif-${CGIF_VERSION} && \
  meson build --prefix=/usr/local --libdir=/usr/local/lib && \
  cd build && \
  ninja && \
  ninja install && \
  cd /tmp && \
  curl -fsSLO https://github.com/libvips/libvips/releases/download/v${VIPS_VERSION}/vips-${VIPS_VERSION}.tar.gz && \
  tar zvxf vips-${VIPS_VERSION}.tar.gz && \
  cd /tmp/vips-${VIPS_VERSION} && \
	CFLAGS="-g -O3" CXXFLAGS="-D_GLIBCXX_USE_CXX11_ABI=0 -g -O3" \
    ./configure \
    --disable-debug \
    --disable-dependency-tracking \
    --disable-introspection \
    --disable-static \
    --enable-gtk-doc-html=no \
    --enable-gtk-doc=no \
    --enable-pyvips8=no && \
  make && \
  make install && \
  ldconfig

WORKDIR ${GOPATH}/src/github.com/cshum/imagor

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN if [ "$RUN_TEST" = 1 ]; then go test ./...; fi
RUN go build -o ${GOPATH}/bin/imagor ./cmd/imagor/main.go

RUN rm -rf /usr/local/lib/python*

FROM debian:bullseye-slim
LABEL maintainer="adrian@cshum.com"

COPY --from=builder /usr/local/lib /usr/local/lib
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /etc/apt/sources.list.d/backports.list /etc/apt/sources.list.d/backports.list

# Install runtime dependencies
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  procps libglib2.0-0 libjpeg62-turbo libpng16-16 libopenexr25 \
  libwebp6 libwebpmux3 libwebpdemux2 libtiff5 libexif12 libxml2 libpoppler-glib8 \
  libpango1.0-0 libmatio11 libopenslide0 libjemalloc2 \
  libgsf-1-114 fftw3 liborc-0.4-0 librsvg2-2 libcfitsio9 libimagequant0 libaom3 libheif1 && \
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
