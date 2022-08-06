// https://libvips.github.io/libvips/API/current/libvips-colour.html

#include <stdlib.h>
#include <vips/vips.h>

int to_colorspace(VipsImage *in, VipsImage **out, VipsInterpretation space);
