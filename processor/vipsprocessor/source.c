#include "source.h"

static gint64 go_read(VipsSourceCustom *source_custom, void *buffer, gint64 length, void* ptr)
{
  return goSourceRead(ptr, buffer, length);
}

static gint64 go_seek(VipsSourceCustom *source_custom, gint64 offset, int whence, void* ptr)
{
  return goSourceSeek(ptr, offset, whence);
}

VipsSourceCustom * create_go_custom_source(void* ptr)
{
  VipsSourceCustom * source_custom = vips_source_custom_new();
  g_signal_connect(source_custom, "read", G_CALLBACK(go_read), ptr);
  g_signal_connect(source_custom, "seek", G_CALLBACK(go_seek), ptr);
  return source_custom;
}

int image_new_from_source(VipsSourceCustom *source, VipsImage **out) {
  *out = vips_image_new_from_source((VipsSource*) source, "", NULL);
  if (!*out) return 1;
  return 0;
}

int image_new_from_source_with_option(VipsSourceCustom *source, VipsImage **out, const char *option_string) {
  *out = vips_image_new_from_source((VipsSource*) source, option_string, NULL);
  if (!*out) return 1;
  return 0;
}

void clear_source(VipsSourceCustom **source_custom) {
  if (G_IS_OBJECT(*source_custom)) g_clear_object(source_custom);
}

