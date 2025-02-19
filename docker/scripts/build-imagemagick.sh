#!/usr/bin/env bash

set -e

# Exit early if ImageMagick is not enabled
if [[ "${IMAGE_MAGICK_ENABLED}" != "1" ]]; then
    echo "Skipping ImageMagick build..."
    exit 0
fi

# libltdl-dev is needed for ImageMagick build process
DEBIAN_FRONTEND=noninteractive \
  apt-get install --no-install-recommends -y \
  libltdl-dev

t=$(mktemp)
wget 'https://dist.1-2.dev/imei.sh' -qO "$t"
bash "$t" --imagemagick-version $IMAGE_MAGICK_VERSION
rm "$t"
cat /var/log/imei-*