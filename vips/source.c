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
  return source_custom;
}

VipsSourceCustom * create_go_custom_source_with_seek(void* ptr)
{
  VipsSourceCustom * source_custom = vips_source_custom_new();
  g_signal_connect(source_custom, "read", G_CALLBACK(go_read), ptr);
  g_signal_connect(source_custom, "seek", G_CALLBACK(go_seek), ptr);
  return source_custom;
}

