#!/usr/bin/env bash

set -e

# If ImageMagick is not included, then we don't need to build libraw. Libvips doesn't use it
if [[ "${IMAGE_MAGICK_ENABLED}" != "1" ]]; then
    echo "Skipping libraw build because ImageMagick is not required..."
    exit 0
fi

# We build libraw from scratch because the debian bookworm package is too old
cd /tmp
curl -fsSLO https://github.com/LibRaw/LibRaw/archive/refs/tags/${LIBRAW_VERSION}.tar.gz
tar xf ${LIBRAW_VERSION}.tar.gz
cd LibRaw-${LIBRAW_VERSION}
ls -la

# libtool is needed for libraw build process
DEBIAN_FRONTEND=noninteractive \
  apt-get install --no-install-recommends -y \
  libtool

# https://github.com/LibRaw/LibRaw/blob/master/INSTALL
autoreconf --install
./configure
make install
cd .. && rm -rf libraw
#ldconfig /usr/local/lib
ldconfig