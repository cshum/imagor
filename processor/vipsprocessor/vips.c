#include "vips.h"
#include <unistd.h>

int image_new_from_file(const char *name, VipsImage **out) {
  *out = vips_image_new_from_file(name, NULL);
  if (!*out) return 1;
  return 0;
}

int image_new_from_buffer(const void *buf, size_t len, VipsImage **out) {
  *out = vips_image_new_from_buffer(buf, len, "", NULL);
  if (!*out) return 1;
  return 0;
}

int image_new_from_buffer_with_option(const void *buf, size_t len, VipsImage **out, const char *option_string) {
  *out = vips_image_new_from_buffer(buf, len, option_string, NULL);
  if (!*out) return 1;
  return 0;
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

void clear_image(VipsImage **image) {
  // https://developer.gnome.org/gobject/stable/gobject-The-Base-Object-Type.html#g-clear-object
  if (G_IS_OBJECT(*image)) g_clear_object(image);
}

int has_alpha_channel(VipsImage *image) { return vips_image_hasalpha(image); }

int copy_image(VipsImage *in, VipsImage **out) {
  return vips_copy(in, out, NULL);
}

int embed_image(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, int extend) {
  return vips_embed(in, out, left, top, width, height, "extend", extend, NULL);
}

int embed_image_background(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, double r, double g, double b, double a) {

  double background[3] = {r, g, b};
  double backgroundRGBA[4] = {r, g, b, a};

  VipsArrayDouble *vipsBackground;

  if (in->Bands <= 3) {
    vipsBackground = vips_array_double_new(background, 3);
  } else {
    vipsBackground = vips_array_double_new(backgroundRGBA, 4);
  }

  int code = vips_embed(in, out, left, top, width, height,
    "extend", VIPS_EXTEND_BACKGROUND, "background", vipsBackground, NULL);

  vips_area_unref(VIPS_AREA(vipsBackground));
  return code;
}

int embed_multi_page_image(VipsImage *in, VipsImage **out, int left, int top, int width,
                         int height, int extend) {
  VipsObject *base = VIPS_OBJECT(vips_image_new());
  int page_height = vips_image_get_page_height(in);
  int in_width = in->Xsize;
  int n_pages = in->Ysize / page_height;

  VipsImage **page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **copy = (VipsImage **) vips_object_local_array(base, 1);

  // split image into cropped frames
  for (int i = 0; i < n_pages; i++) {
    if (
      vips_extract_area(in, &page[i], 0, page_height * i, in_width, page_height, NULL) ||
      vips_embed(page[i], &page[i], left, top, width, height, "extend", extend, NULL)
    ) {
      g_object_unref(base);
      return -1;
    }
  }
  // reassemble frames and set page height
  // copy before modifying metadata
  if(
    vips_arrayjoin(page, &copy[0], n_pages, "across", 1, NULL) ||
    vips_copy(copy[0], out, NULL)
  ) {
    g_object_unref(base);
    return -1;
  }
  vips_image_set_int(*out, VIPS_META_PAGE_HEIGHT, height);
  g_object_unref(base);
  return 0;
}

int embed_multi_page_image_background(VipsImage *in, VipsImage **out, int left, int top, int width,
                                   int height, double r, double g, double b, double a) {
  double background[3] = {r, g, b};
  double backgroundRGBA[4] = {r, g, b, a};

  VipsArrayDouble *vipsBackground;

  if (in->Bands <= 3) {
    vipsBackground = vips_array_double_new(background, 3);
  } else {
    vipsBackground = vips_array_double_new(backgroundRGBA, 4);
  }
  VipsObject *base = VIPS_OBJECT(vips_image_new());
  int page_height = vips_image_get_page_height(in);
  int in_width = in->Xsize;
  int n_pages = in->Ysize / page_height;

  VipsImage **page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **copy = (VipsImage **) vips_object_local_array(base, 1);

  // split image into cropped frames
  for (int i = 0; i < n_pages; i++) {
    if (
      vips_extract_area(in, &page[i], 0, page_height * i, in_width, page_height, NULL) ||
      vips_embed(page[i], &page[i], left, top, width, height,
          "extend", VIPS_EXTEND_BACKGROUND, "background", vipsBackground, NULL)
    ) {
      vips_area_unref(VIPS_AREA(vipsBackground));
      g_object_unref(base);
      return -1;
    }
  }
  // reassemble frames and set page height
  // copy before modifying metadata
  if(
    vips_arrayjoin(page, &copy[0], n_pages, "across", 1, NULL) ||
    vips_copy(copy[0], out, NULL)
  ) {
    vips_area_unref(VIPS_AREA(vipsBackground));
    g_object_unref(base);
    return -1;
  }
  vips_image_set_int(*out, VIPS_META_PAGE_HEIGHT, height);
  vips_area_unref(VIPS_AREA(vipsBackground));
  g_object_unref(base);
  return 0;
}

int flip_image(VipsImage *in, VipsImage **out, int direction) {
  return vips_flip(in, out, direction, NULL);
}

int extract_image_area(VipsImage *in, VipsImage **out, int left, int top,
                       int width, int height) {
  return vips_extract_area(in, out, left, top, width, height, NULL);
}

int extract_area_multi_page(VipsImage *in, VipsImage **out, int left, int top, int width, int height) {
  VipsObject *base = VIPS_OBJECT(vips_image_new());
  int page_height = vips_image_get_page_height(in);
  int n_pages = in->Ysize / page_height;

  VipsImage **page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **copy = (VipsImage **) vips_object_local_array(base, 1);

  // split image into cropped frames
  for (int i = 0; i < n_pages; i++) {
    if(vips_extract_area(in, &page[i], left, page_height * i + top, width, height, NULL)) {
      g_object_unref(base);
      return -1;
    }
  }
  // reassemble frames and set page height
  // copy before modifying metadata
  if(
    vips_arrayjoin(page, &copy[0], n_pages, "across", 1, NULL) ||
    vips_copy(copy[0], out, NULL)
  ) {
    g_object_unref(base);
    return -1;
  }
  vips_image_set_int(*out, VIPS_META_PAGE_HEIGHT, height);
  g_object_unref(base);
  return 0;
}

int rotate_image(VipsImage *in, VipsImage **out, VipsAngle angle) {
  return vips_rot(in, out, angle, NULL);
}

int rotate_image_multi_page(VipsImage *in, VipsImage **out, VipsAngle angle) {
  VipsObject *base = VIPS_OBJECT(vips_image_new());
  int page_height = vips_image_get_page_height(in);
  int in_width = in->Xsize;
  int n_pages = in->Ysize / page_height;

  VipsImage **page = (VipsImage **) vips_object_local_array(base, n_pages);
  VipsImage **copy = (VipsImage **) vips_object_local_array(base, 1);

  // split image into cropped frames
  for (int i = 0; i < n_pages; i++) {
    if (
      vips_extract_area(in, &page[i], 0, page_height * i, in_width, page_height, NULL) ||
      vips_rot(page[i], &page[i], angle, NULL)
    ) {
      g_object_unref(base);
      return -1;
    }
  }
  // reassemble frames and set page height if rotate 90 or 270
  // copy before modifying metadata
  if(
    vips_arrayjoin(page, &copy[0], n_pages, "across", 1, NULL) ||
    vips_copy(copy[0], out, NULL)
  ) {
    g_object_unref(base);
    return -1;
  }
  if (angle == VIPS_ANGLE_D90 || angle == VIPS_ANGLE_D270) {
    vips_image_set_int(*out, VIPS_META_PAGE_HEIGHT, in_width);
  }
  g_object_unref(base);
  return 0;
}

int flatten_image(VipsImage *in, VipsImage **out, double r, double g,
                  double b) {
  if (is_16bit(in->Type)) {
    r = 65535 * r / 255;
    g = 65535 * g / 255;
    b = 65535 * b / 255;
  }

  double background[3] = {r, g, b};
  VipsArrayDouble *vipsBackground = vips_array_double_new(background, 3);

  int code = vips_flatten(in, out, "background", vipsBackground, "max_alpha",
                          is_16bit(in->Type) ? 65535.0 : 255.0, NULL);

  vips_area_unref(VIPS_AREA(vipsBackground));
  return code;
}

int is_16bit(VipsInterpretation interpretation) {
  return interpretation == VIPS_INTERPRETATION_RGB16 ||
         interpretation == VIPS_INTERPRETATION_GREY16;
}

int add_alpha(VipsImage *in, VipsImage **out) {
  return vips_addalpha(in, out, NULL);
}

double max_alpha(VipsImage *in) {
  switch (in->BandFmt) {
    case VIPS_FORMAT_USHORT:
      return 65535;
    case VIPS_FORMAT_FLOAT:
    case VIPS_FORMAT_DOUBLE:
      return 1.0;
    default:
      return 255;
  }
}

int composite2_image(VipsImage *base, VipsImage *overlay, VipsImage **out,
                     int mode, gint x, gint y) {
  return vips_composite2(base, overlay, out, mode, "x", x, "y", y, NULL);
}

int replicate(VipsImage *in, VipsImage **out, int across, int down) {
  return vips_replicate(in, out, across, down, NULL);
}

int linear(VipsImage *in, VipsImage **out, double *a, double *b, int n) {
  return vips_linear(in, out, a, b, n, NULL);
}

int find_trim(VipsImage *in, int *left, int *top, int *width, int *height,
              double threshold, double r, double g, double b) {

  if (in->Type == VIPS_INTERPRETATION_RGB16 || in->Type == VIPS_INTERPRETATION_GREY16) {
    r = 65535 * r / 255;
    g = 65535 * g / 255;
    b = 65535 * b / 255;
  }

  double background[3] = {r, g, b};
  VipsArrayDouble *vipsBackground = vips_array_double_new(background, 3);

  int code = vips_find_trim(in, left, top, width, height, "threshold", threshold, "background", vipsBackground, NULL);

  vips_area_unref(VIPS_AREA(vipsBackground));
  return code;
}

int getpoint(VipsImage *in, double **vector, int n, int x, int y) {
  return vips_getpoint(in, vector, &n, x, y, NULL);
}

int to_colorspace(VipsImage *in, VipsImage **out, VipsInterpretation space) {
  return vips_colourspace(in, out, space, NULL);
}

int gaussian_blur_image(VipsImage *in, VipsImage **out, double sigma) {
  return vips_gaussblur(in, out, sigma, NULL);
}

int sharpen_image(VipsImage *in, VipsImage **out, double sigma, double x1,
                  double m2) {
  return vips_sharpen(in, out, "sigma", sigma, "x1", x1, "m2", m2, NULL);
}

gboolean remove_icc_profile(VipsImage *in) {
  return vips_image_remove(in, VIPS_META_ICC_NAME);
}

int get_meta_orientation(VipsImage *in) {
  int orientation = 0;
  if (vips_image_get_typeof(in, VIPS_META_ORIENTATION) != 0) {
    vips_image_get_int(in, VIPS_META_ORIENTATION, &orientation);
  }

  return orientation;
}

// https://libvips.github.io/libvips/API/current/libvips-header.html#vips-image-get-n-pages
int get_image_n_pages(VipsImage *in) {
  int n_pages = 0;
  n_pages = vips_image_get_n_pages(in);
  return n_pages;
}

// https://www.libvips.org/API/current/libvips-header.html#vips-image-get-page-height
int get_page_height(VipsImage *in) {
  int page_height = 0;
  page_height = vips_image_get_page_height(in);
  return page_height;
}

void set_page_height(VipsImage *in, int height) {
  vips_image_set_int(in, VIPS_META_PAGE_HEIGHT, height);
}

int get_meta_loader(const VipsImage *in, const char **out) {
  return vips_image_get_string(in, VIPS_META_LOADER, out);
}

void set_image_delay(VipsImage *in, const int *array, int n) {
  return vips_image_set_array_int(in, "delay", array, n);
}
