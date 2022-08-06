// https://libvips.github.io/libvips/API/current/VipsImage.html

#include <stdlib.h>
#include <vips/vips.h>

int has_alpha_channel(VipsImage *image);

void clear_image(VipsImage **image);

int image_new_from_buffer_with_option(const void *buf, size_t len, VipsImage **out, const char *option_string);

int image_new_from_buffer(const void *buf, size_t len, VipsImage **out);

int image_new_from_file(const char *name, VipsImage **out);
