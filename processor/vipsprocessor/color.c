#include "color.h"
#include <unistd.h>

int is_colorspace_supported(VipsImage *in) {
  return vips_colourspace_issupported(in) ? 1 : 0;
}

int to_colorspace(VipsImage *in, VipsImage **out, VipsInterpretation space) {
  return vips_colourspace(in, out, space, NULL);
}
