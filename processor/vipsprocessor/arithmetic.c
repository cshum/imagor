#include "arithmetic.h"

int linear(VipsImage *in, VipsImage **out, double *a, double *b, int n) {
  return vips_linear(in, out, a, b, n, NULL);
}

int find_trim(VipsImage *in, int *left, int *top, int *width, int *height,
              double threshold, double r, double g, double b) {

  if (in->Type == VIPS_INTERPRETATION_RGB16 || in->Type == VIPS_INTERPRETATION_GREY16) {
    r = 65535 * r / 255;
    g = 65535 * g / 255;
    b = 65535 * b / 255;
  }

  double background[3] = {r, g, b};
  VipsArrayDouble *vipsBackground = vips_array_double_new(background, 3);

  int code = vips_find_trim(in, left, top, width, height, "threshold", threshold, "background", vipsBackground, NULL);

  vips_area_unref(VIPS_AREA(vipsBackground));
  return code;
}

int getpoint(VipsImage *in, double **vector, int n, int x, int y) {
  return vips_getpoint(in, vector, &n, x, y, NULL);
}
