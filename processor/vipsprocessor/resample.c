#include "resample.h"

int thumbnail(const char *filename, VipsImage **out,
                    int width, int height, int crop, int size) {
  return vips_thumbnail(filename, out, width, "height", height,
                              "crop", crop, "size", size, NULL);
}

int thumbnail_image(VipsImage *in, VipsImage **out, int width, int height,
                    int crop, int size) {
  return vips_thumbnail_image(in, out, width, "height", height, "crop", crop,
                              "size", size, NULL);
}

int thumbnail_buffer_with_option(void *buf, size_t len, VipsImage **out,
                    int width, int height, int crop, int size,
                    const char *option_string) {
  return vips_thumbnail_buffer(buf, len, out, width, "height", height,
                              "crop", crop, "size", size,
                              "option_string", option_string, NULL);
}

int thumbnail_buffer(void *buf, size_t len, VipsImage **out,
                    int width, int height, int crop, int size) {
  return vips_thumbnail_buffer(buf, len, out, width, "height", height,
                              "crop", crop, "size", size, NULL);
}
