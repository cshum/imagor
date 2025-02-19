#!/usr/bin/env bash

set -e

cd /tmp
curl -fsSLO https://github.com/libvips/libvips/releases/download/v${VIPS_VERSION}/vips-${VIPS_VERSION}.tar.xz
tar xf vips-${VIPS_VERSION}.tar.xz
cd vips-${VIPS_VERSION}

# Determine whether to enable or disable ImageMagick support
if [[ "${IMAGE_MAGICK_ENABLED}" == "1" ]]; then
    MAGICK_OPTION="enabled"
else
    MAGICK_OPTION="disabled"
fi

meson setup _build \
--buildtype=release \
--strip \
--prefix=/usr/local \
--libdir=lib \
-Dgtk_doc=false \
-Dmagick=${MAGICK_OPTION} \
-Dintrospection=disabled
ninja -C _build
ninja -C _build install
ldconfig
rm -rf /usr/local/lib/libvips-cpp.*
rm -rf /usr/local/lib/*.a
rm -rf /usr/local/lib/*.la