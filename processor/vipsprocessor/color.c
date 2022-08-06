#include "color.h"
#include <unistd.h>

int to_colorspace(VipsImage *in, VipsImage **out, VipsInterpretation space) {
  return vips_colourspace(in, out, space, NULL);
}
