#include <stdlib.h>
#include <vips/vips.h>

int image_new_from_file(const char *name, VipsImage **out);

int image_new_from_buffer(const void *buf, size_t len, VipsImage **out);

int image_new_from_buffer_with_option(const void *buf, size_t len, VipsImage **out, const char *option_string);

int thumbnail_image(VipsImage *in, VipsImage **out, int width, int height,
                    int crop, int size);
int thumbnail_buffer(void *buf, size_t len, VipsImage **out, int width, int height,
                    int crop, int size);
int thumbnail_buffer_with_option(void *buf, size_t len, VipsImage **out,
                    int width, int height, int crop, int size,
                    const char *option_string);

int has_alpha_channel(VipsImage *image);

void clear_image(VipsImage **image);

int copy_image(VipsImage *in, VipsImage **out);

int embed_image(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, int extend);
int embed_image_background(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, double r, double g, double b, double a);
int embed_multi_page_image(VipsImage *in, VipsImage **out, int left, int top, int width,
                int height, int extend);
int embed_multi_page_image_background(VipsImage *in, VipsImage **out, int left, int top,
                int width, int height, double r, double g, double b, double a);

int flip_image(VipsImage *in, VipsImage **out, int direction);

int extract_image_area(VipsImage *in, VipsImage **out, int left, int top,
                       int width, int height);
int extract_area_multi_page(VipsImage *in, VipsImage **out, int left, int top,
                       int width, int height);

int rotate_image(VipsImage *in, VipsImage **out, VipsAngle angle);
int rotate_image_multi_page(VipsImage *in, VipsImage **out, VipsAngle angle);
int flatten_image(VipsImage *in, VipsImage **out, double r, double g, double b);
int add_alpha(VipsImage *in, VipsImage **out);
double max_alpha(VipsImage *in);

int composite2_image(VipsImage *base, VipsImage *overlay, VipsImage **out,
                     int mode, gint x, gint y);

int is_16bit(VipsInterpretation interpretation);

int replicate(VipsImage *in, VipsImage **out, int across, int down);


int linear(VipsImage *in, VipsImage **out, double *a, double *b, int n);
int find_trim(VipsImage *in, int *left, int *top, int *width, int *height,
              double threshold, double r, double g, double b);
int getpoint(VipsImage *in, double **vector, int n, int x, int y);

int to_colorspace(VipsImage *in, VipsImage **out, VipsInterpretation space);

int gaussian_blur_image(VipsImage *in, VipsImage **out, double sigma);
int sharpen_image(VipsImage *in, VipsImage **out, double sigma, double x1,
                  double m2);

int remove_icc_profile(VipsImage *in);

int get_meta_orientation(VipsImage *in);
int get_image_n_pages(VipsImage *in);
int get_page_height(VipsImage *in);
void set_page_height(VipsImage *in, int height);
int get_meta_loader(const VipsImage *in, const char **out);
void set_image_delay(VipsImage *in, const int *array, int n);
