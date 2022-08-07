package vipsprocessor

// #include "foreign.h"
import "C"
import (
	"strings"
	"unsafe"
)

// ImageType represents an image type
type ImageType int

// ImageType enum
const (
	ImageTypeUnknown ImageType = C.UNKNOWN
	ImageTypeGIF     ImageType = C.GIF
	ImageTypeJPEG    ImageType = C.JPEG
	ImageTypeMagick  ImageType = C.MAGICK
	ImageTypePDF     ImageType = C.PDF
	ImageTypePNG     ImageType = C.PNG
	ImageTypeSVG     ImageType = C.SVG
	ImageTypeTIFF    ImageType = C.TIFF
	ImageTypeWEBP    ImageType = C.WEBP
	ImageTypeHEIF    ImageType = C.HEIF
	ImageTypeBMP     ImageType = C.BMP
	ImageTypeAVIF    ImageType = C.AVIF
	ImageTypeJP2K    ImageType = C.JP2K
)

// ImageTypes defines the various image types supported by govips
var ImageTypes = map[ImageType]string{
	ImageTypeGIF:    "gif",
	ImageTypeJPEG:   "jpeg",
	ImageTypeMagick: "magick",
	ImageTypePDF:    "pdf",
	ImageTypePNG:    "png",
	ImageTypeSVG:    "svg",
	ImageTypeTIFF:   "tiff",
	ImageTypeWEBP:   "webp",
	ImageTypeHEIF:   "heif",
	ImageTypeBMP:    "bmp",
	ImageTypeAVIF:   "avif",
	ImageTypeJP2K:   "jp2k",
}

var imageTypeMap = map[string]ImageType{
	"gif":    ImageTypeGIF,
	"jpeg":   ImageTypeJPEG,
	"jpg":    ImageTypeJPEG,
	"magick": ImageTypeMagick,
	"pdf":    ImageTypePDF,
	"png":    ImageTypePNG,
	"svg":    ImageTypeSVG,
	"tiff":   ImageTypeTIFF,
	"webp":   ImageTypeWEBP,
	"heif":   ImageTypeHEIF,
	"bmp":    ImageTypeBMP,
	"avif":   ImageTypeAVIF,
	"jp2":    ImageTypeJP2K,
}

var imageMimeTypeMap = map[string]string{
	"gif":  "image/gif",
	"jpeg": "image/jpeg",
	"jpg":  "image/jpeg",
	"pdf":  "application/pdf",
	"png":  "image/png",
	"svg":  "image/svg+xml",
	"tiff": "image/tiff",
	"webp": "image/webp",
	"heif": "image/heif",
	"bmp":  "image/bmp",
	"avif": "image/avif",
	"jp2":  "image/jp2",
}

// vipsDetermineImageTypeFromMetaLoader determine the image type from vips-loader metadata
func vipsDetermineImageTypeFromMetaLoader(in *C.VipsImage) ImageType {
	if in != nil {
		if vipsLoader, ok := vipsImageGetMetaLoader(in); ok {
			if strings.HasPrefix(vipsLoader, "jpeg") {
				return ImageTypeJPEG
			}
			if strings.HasPrefix(vipsLoader, "png") {
				return ImageTypePNG
			}
			if strings.HasPrefix(vipsLoader, "gif") {
				return ImageTypeGIF
			}
			if strings.HasPrefix(vipsLoader, "svg") {
				return ImageTypeSVG
			}
			if strings.HasPrefix(vipsLoader, "webp") {
				return ImageTypeWEBP
			}
			if strings.HasPrefix(vipsLoader, "avif") {
				return ImageTypeAVIF
			}
			if strings.HasPrefix(vipsLoader, "heif") {
				return ImageTypeHEIF
			}
			if strings.HasPrefix(vipsLoader, "tiff") {
				return ImageTypeTIFF
			}
			if strings.HasPrefix(vipsLoader, "pdf") {
				return ImageTypePDF
			}
			if strings.HasPrefix(vipsLoader, "jp2k") {
				return ImageTypeJP2K
			}
			if strings.HasPrefix(vipsLoader, "magick") {
				return ImageTypeMagick
			}
		}
	}
	return ImageTypeUnknown
}

func vipsSaveJPEGToBuffer(in *C.VipsImage, params JpegExportParams) ([]byte, error) {
	p := C.create_save_params(C.JPEG)
	p.inputImage = in
	p.stripMetadata = C.int(boolToInt(params.StripMetadata))
	p.quality = C.int(params.Quality)
	p.interlace = C.int(boolToInt(params.Interlace))
	p.jpegOptimizeCoding = C.int(boolToInt(params.OptimizeCoding))
	p.jpegSubsample = C.VipsForeignJpegSubsample(params.SubsampleMode)
	p.jpegTrellisQuant = C.int(boolToInt(params.TrellisQuant))
	p.jpegOvershootDeringing = C.int(boolToInt(params.OvershootDeringing))
	p.jpegOptimizeScans = C.int(boolToInt(params.OptimizeScans))
	p.jpegQuantTable = C.int(params.QuantTable)

	return vipsSaveToBuffer(p)
}

func vipsSavePNGToBuffer(in *C.VipsImage, params PngExportParams) ([]byte, error) {
	p := C.create_save_params(C.PNG)
	p.inputImage = in
	p.quality = C.int(params.Quality)
	p.stripMetadata = C.int(boolToInt(params.StripMetadata))
	p.interlace = C.int(boolToInt(params.Interlace))
	p.pngCompression = C.int(params.Compression)
	p.pngFilter = C.VipsForeignPngFilter(params.Filter)
	p.pngPalette = C.int(boolToInt(params.Palette))
	p.pngDither = C.double(params.Dither)
	p.pngBitdepth = C.int(params.Bitdepth)

	return vipsSaveToBuffer(p)
}

func vipsSaveWebPToBuffer(in *C.VipsImage, params WebpExportParams) ([]byte, error) {
	p := C.create_save_params(C.WEBP)
	p.inputImage = in
	p.stripMetadata = C.int(boolToInt(params.StripMetadata))
	p.quality = C.int(params.Quality)
	p.webpLossless = C.int(boolToInt(params.Lossless))
	p.webpNearLossless = C.int(boolToInt(params.NearLossless))
	p.webpReductionEffort = C.int(params.ReductionEffort)

	if params.IccProfile != "" {
		p.webpIccProfile = C.CString(params.IccProfile)
		defer C.free(unsafe.Pointer(p.webpIccProfile))
	}

	return vipsSaveToBuffer(p)
}

func vipsSaveTIFFToBuffer(in *C.VipsImage, params TiffExportParams) ([]byte, error) {
	p := C.create_save_params(C.TIFF)
	p.inputImage = in
	p.stripMetadata = C.int(boolToInt(params.StripMetadata))
	p.quality = C.int(params.Quality)
	p.tiffCompression = C.VipsForeignTiffCompression(params.Compression)

	return vipsSaveToBuffer(p)
}

func vipsSaveHEIFToBuffer(in *C.VipsImage, params HeifExportParams) ([]byte, error) {
	p := C.create_save_params(C.HEIF)
	p.inputImage = in
	p.outputFormat = C.HEIF
	p.quality = C.int(params.Quality)
	p.heifLossless = C.int(boolToInt(params.Lossless))

	return vipsSaveToBuffer(p)
}

func vipsSaveAVIFToBuffer(in *C.VipsImage, params AvifExportParams) ([]byte, error) {
	p := C.create_save_params(C.AVIF)
	p.inputImage = in
	p.outputFormat = C.AVIF
	p.quality = C.int(params.Quality)
	p.heifLossless = C.int(boolToInt(params.Lossless))
	p.avifSpeed = C.int(params.Speed)

	return vipsSaveToBuffer(p)
}

func vipsSaveJP2KToBuffer(in *C.VipsImage, params Jp2kExportParams) ([]byte, error) {
	p := C.create_save_params(C.JP2K)
	p.inputImage = in
	p.outputFormat = C.JP2K
	p.quality = C.int(params.Quality)
	p.jp2kLossless = C.int(boolToInt(params.Lossless))
	p.jp2kTileWidth = C.int(params.TileWidth)
	p.jp2kTileHeight = C.int(params.TileHeight)
	p.jpegSubsample = C.VipsForeignJpegSubsample(params.SubsampleMode)

	return vipsSaveToBuffer(p)
}

func vipsSaveGIFToBuffer(in *C.VipsImage, params GifExportParams) ([]byte, error) {
	p := C.create_save_params(C.GIF)
	p.inputImage = in
	p.quality = C.int(params.Quality)
	p.gifDither = C.double(params.Dither)
	p.gifEffort = C.int(params.Effort)
	p.gifBitdepth = C.int(params.Bitdepth)

	return vipsSaveToBuffer(p)
}

func vipsSaveToBuffer(params C.struct_SaveParams) ([]byte, error) {
	if err := C.save_to_buffer(&params); err != 0 {
		if params.outputBuffer != nil {
			gFreePointer(params.outputBuffer)
		}
		return nil, handleVipsError()
	}

	buf := C.GoBytes(params.outputBuffer, C.int(params.outputLen))
	defer gFreePointer(params.outputBuffer)

	return buf, nil
}
