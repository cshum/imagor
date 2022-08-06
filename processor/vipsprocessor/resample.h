// https://libvips.github.io/libvips/API/current/libvips-resample.html

#include <stdlib.h>
#include <vips/vips.h>

int thumbnail_image(VipsImage *in, VipsImage **out, int width, int height,
                    int crop, int size);
int thumbnail_buffer(void *buf, size_t len, VipsImage **out, int width, int height,
                    int crop, int size);
int thumbnail_buffer_with_option(void *buf, size_t len, VipsImage **out,
                    int width, int height, int crop, int size,
                    const char *option_string);
