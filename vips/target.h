#include <stdlib.h>
#include <vips/vips.h>

extern long long goTargetWrite(void*, const void *, long long);

extern long long goTargetFinish(void*);

static gint64 go_write(VipsTargetCustom *target_custom, const void *data, gint64 length, void* ptr);

static void go_finish(VipsTargetCustom *target_custom, void* ptr);

VipsTargetCustom * create_go_custom_target(void* ptr);

void clear_target(VipsTargetCustom **target_custom);
