#include <stdlib.h>
#include <vips/vips.h>

extern long long goSourceRead(void*, void *, long long);

extern long long goSourceSeek(void*, long long, int);

static gint64 go_read(VipsSourceCustom *source_custom, void *buffer, gint64 length, void* ptr);

static gint64 go_seek(VipsSourceCustom *source_custom, gint64 offset, int whence, void* ptr);

VipsSourceCustom * create_go_custom_source(void* ptr);
VipsSourceCustom * create_go_custom_source_with_seek(void* ptr);

void clear_source(VipsSourceCustom **source_custom);