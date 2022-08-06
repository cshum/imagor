#include "image.h"

int has_alpha_channel(VipsImage *image) { return vips_image_hasalpha(image); }

void clear_image(VipsImage **image) {
  // https://developer.gnome.org/gobject/stable/gobject-The-Base-Object-Type.html#g-clear-object
  if (G_IS_OBJECT(*image)) g_clear_object(image);
}

int image_new_from_buffer_with_option(const void *buf, size_t len, VipsImage **out, const char *option_string) {
  *out = vips_image_new_from_buffer(buf, len, option_string, NULL);
  if (!*out) return 1;
  return 0;
}

int image_new_from_buffer(const void *buf, size_t len, VipsImage **out) {
  *out = vips_image_new_from_buffer(buf, len, "", NULL);
  if (!*out) return 1;
  return 0;
}

int image_new_from_file(const char *name, VipsImage **out) {
  *out = vips_image_new_from_file(name, NULL);
  if (!*out) return 1;
  return 0;
}

