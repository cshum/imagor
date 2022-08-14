#include "target.h"

static gint64 go_write(VipsTargetCustom *target_custom, const void *data, gint64 length, void* ptr)
{
	return goTargetWrite(ptr, data, length);
}

static void go_finish(VipsTargetCustom *target_custom, void* ptr)
{
	goTargetFinish(ptr);
}

VipsTargetCustom * create_go_custom_target(void* ptr)
{
	VipsTargetCustom * target_custom = vips_target_custom_new();
	g_signal_connect(target_custom, "write", G_CALLBACK(go_write), ptr);
	g_signal_connect(target_custom, "finish", G_CALLBACK(go_finish), ptr);
	return target_custom;
}

void clear_target(VipsTargetCustom **target_custom) {
  if (G_IS_OBJECT(*target_custom)) g_clear_object(target_custom);
}
