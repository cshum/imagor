// https://libvips.github.io/libvips/API/current/VipsForeignSave.html

// clang-format off
// include order matters
#include <stdlib.h>

#include <vips/vips.h>
#include <vips/foreign.h>
// clang-format n

#ifndef BOOL
#define BOOL int
#endif

typedef enum types {
  UNKNOWN = 0,
  JPEG,
  WEBP,
  PNG,
  TIFF,
  GIF,
  PDF,
  SVG,
  MAGICK,
  HEIF,
  BMP,
  AVIF,
  JP2K
} ImageType;

typedef struct SaveParams {
  VipsImage *inputImage;
  void *outputBuffer;
  ImageType outputFormat;
  size_t outputLen;

  BOOL stripMetadata;
  int quality;
  BOOL interlace;

  // JPEG
  BOOL jpegOptimizeCoding;
  VipsForeignJpegSubsample jpegSubsample;
  BOOL jpegTrellisQuant;
  BOOL jpegOvershootDeringing;
  BOOL jpegOptimizeScans;
  int jpegQuantTable;

  // PNG
  int pngCompression;
  VipsForeignPngFilter pngFilter;
  BOOL pngPalette;
  double pngDither;
  int pngBitdepth;

  // GIF (with CGIF)
  double gifDither;
  int gifEffort;
  int gifBitdepth;

  // WEBP
  BOOL webpLossless;
  BOOL webpNearLossless;
  int webpReductionEffort;
  char *webpIccProfile;

  // HEIF
  BOOL heifLossless;

  // TIFF
  VipsForeignTiffCompression tiffCompression;
  VipsForeignTiffPredictor tiffPredictor;
  BOOL tiffPyramid;
  BOOL tiffTile;
  int tiffTileHeight;
  int tiffTileWidth;
  double tiffXRes;
  double tiffYRes;

  // AVIF
  int avifSpeed;

  // JPEG2000
  BOOL jp2kLossless;
  int jp2kTileWidth;
  int	jp2kTileHeight;
} SaveParams;

SaveParams create_save_params(ImageType outputFormat);
int save_to_buffer(SaveParams *params);

