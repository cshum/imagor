// https://libvips.github.io/libvips/API/current/libvips-arithmetic.html

#include <stdlib.h>
#include <vips/vips.h>

int linear(VipsImage *in, VipsImage **out, double *a, double *b, int n);
int find_trim(VipsImage *in, int *left, int *top, int *width, int *height,
              double threshold, double r, double g, double b);
int getpoint(VipsImage *in, double **vector, int n, int x, int y);
