package vips

// #include "vips.h"
import "C"
import "strings"

// ImageType represents an image type
type ImageType int

// ImageType enum
const (
	ImageTypeUnknown ImageType = iota
	ImageTypeGIF
	ImageTypeJPEG
	ImageTypeMagick
	ImageTypePDF
	ImageTypePNG
	ImageTypeSVG
	ImageTypeTIFF
	ImageTypeWEBP
	ImageTypeHEIF
	ImageTypeBMP
	ImageTypeAVIF
	ImageTypeJP2K
)

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

// ImageTypes defines the various image types supported by vips
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

// Color represents an RGB
type Color struct {
	R, G, B uint8
}

// ColorRGBA represents an RGB with alpha channel (A)
type ColorRGBA struct {
	R, G, B, A uint8
}

// Interpretation represents VIPS_INTERPRETATION type
type Interpretation int

// Interpretation enum
const (
	InterpretationError     Interpretation = C.VIPS_INTERPRETATION_ERROR
	InterpretationMultiband Interpretation = C.VIPS_INTERPRETATION_MULTIBAND
	InterpretationBW        Interpretation = C.VIPS_INTERPRETATION_B_W
	InterpretationHistogram Interpretation = C.VIPS_INTERPRETATION_HISTOGRAM
	InterpretationXYZ       Interpretation = C.VIPS_INTERPRETATION_XYZ
	InterpretationLAB       Interpretation = C.VIPS_INTERPRETATION_LAB
	InterpretationCMYK      Interpretation = C.VIPS_INTERPRETATION_CMYK
	InterpretationLABQ      Interpretation = C.VIPS_INTERPRETATION_LABQ
	InterpretationRGB       Interpretation = C.VIPS_INTERPRETATION_RGB
	InterpretationRGB16     Interpretation = C.VIPS_INTERPRETATION_RGB16
	InterpretationCMC       Interpretation = C.VIPS_INTERPRETATION_CMC
	InterpretationLCH       Interpretation = C.VIPS_INTERPRETATION_LCH
	InterpretationLABS      Interpretation = C.VIPS_INTERPRETATION_LABS
	InterpretationSRGB      Interpretation = C.VIPS_INTERPRETATION_sRGB
	InterpretationYXY       Interpretation = C.VIPS_INTERPRETATION_YXY
	InterpretationFourier   Interpretation = C.VIPS_INTERPRETATION_FOURIER
	InterpretationGrey16    Interpretation = C.VIPS_INTERPRETATION_GREY16
	InterpretationMatrix    Interpretation = C.VIPS_INTERPRETATION_MATRIX
	InterpretationScRGB     Interpretation = C.VIPS_INTERPRETATION_scRGB
	InterpretationHSV       Interpretation = C.VIPS_INTERPRETATION_HSV
)

// Intent represents VIPS_INTENT type
type Intent int

//Intent enum
const (
	IntentPerceptual Intent = C.VIPS_INTENT_PERCEPTUAL
	IntentRelative   Intent = C.VIPS_INTENT_RELATIVE
	IntentSaturation Intent = C.VIPS_INTENT_SATURATION
	IntentAbsolute   Intent = C.VIPS_INTENT_ABSOLUTE
	IntentLast       Intent = C.VIPS_INTENT_LAST
)

// BlendMode gives the various Porter-Duff and PDF blend modes.
// See https://libvips.github.io/libvips/API/current/libvips-conversion.html#VipsBlendMode
type BlendMode int

// Constants define the various Porter-Duff and PDF blend modes.
// See https://libvips.github.io/libvips/API/current/libvips-conversion.html#VipsBlendMode
const (
	BlendModeClear      BlendMode = C.VIPS_BLEND_MODE_CLEAR
	BlendModeSource     BlendMode = C.VIPS_BLEND_MODE_SOURCE
	BlendModeOver       BlendMode = C.VIPS_BLEND_MODE_OVER
	BlendModeIn         BlendMode = C.VIPS_BLEND_MODE_IN
	BlendModeOut        BlendMode = C.VIPS_BLEND_MODE_OUT
	BlendModeAtop       BlendMode = C.VIPS_BLEND_MODE_ATOP
	BlendModeDest       BlendMode = C.VIPS_BLEND_MODE_DEST
	BlendModeDestOver   BlendMode = C.VIPS_BLEND_MODE_DEST_OVER
	BlendModeDestIn     BlendMode = C.VIPS_BLEND_MODE_DEST_IN
	BlendModeDestOut    BlendMode = C.VIPS_BLEND_MODE_DEST_OUT
	BlendModeDestAtop   BlendMode = C.VIPS_BLEND_MODE_DEST_ATOP
	BlendModeXOR        BlendMode = C.VIPS_BLEND_MODE_XOR
	BlendModeAdd        BlendMode = C.VIPS_BLEND_MODE_ADD
	BlendModeSaturate   BlendMode = C.VIPS_BLEND_MODE_SATURATE
	BlendModeMultiply   BlendMode = C.VIPS_BLEND_MODE_MULTIPLY
	BlendModeScreen     BlendMode = C.VIPS_BLEND_MODE_SCREEN
	BlendModeOverlay    BlendMode = C.VIPS_BLEND_MODE_OVERLAY
	BlendModeDarken     BlendMode = C.VIPS_BLEND_MODE_DARKEN
	BlendModeLighten    BlendMode = C.VIPS_BLEND_MODE_LIGHTEN
	BlendModeColorDodge BlendMode = C.VIPS_BLEND_MODE_COLOUR_DODGE
	BlendModeColorBurn  BlendMode = C.VIPS_BLEND_MODE_COLOUR_BURN
	BlendModeHardLight  BlendMode = C.VIPS_BLEND_MODE_HARD_LIGHT
	BlendModeSoftLight  BlendMode = C.VIPS_BLEND_MODE_SOFT_LIGHT
	BlendModeDifference BlendMode = C.VIPS_BLEND_MODE_DIFFERENCE
	BlendModeExclusion  BlendMode = C.VIPS_BLEND_MODE_EXCLUSION
)

// Direction represents VIPS_DIRECTION type
type Direction int

// Direction enum
const (
	DirectionHorizontal Direction = C.VIPS_DIRECTION_HORIZONTAL
	DirectionVertical   Direction = C.VIPS_DIRECTION_VERTICAL
)

// Angle represents VIPS_ANGLE type
type Angle int

// Angle enum
const (
	Angle0   Angle = C.VIPS_ANGLE_D0
	Angle90  Angle = C.VIPS_ANGLE_D90
	Angle180 Angle = C.VIPS_ANGLE_D180
	Angle270 Angle = C.VIPS_ANGLE_D270
)

// Angle45 represents VIPS_ANGLE45 type
type Angle45 int

// Angle45 enum
const (
	Angle45_0   Angle45 = C.VIPS_ANGLE45_D0
	Angle45_45  Angle45 = C.VIPS_ANGLE45_D45
	Angle45_90  Angle45 = C.VIPS_ANGLE45_D90
	Angle45_135 Angle45 = C.VIPS_ANGLE45_D135
	Angle45_180 Angle45 = C.VIPS_ANGLE45_D180
	Angle45_225 Angle45 = C.VIPS_ANGLE45_D225
	Angle45_270 Angle45 = C.VIPS_ANGLE45_D270
	Angle45_315 Angle45 = C.VIPS_ANGLE45_D315
)

// ExtendStrategy represents VIPS_EXTEND type
type ExtendStrategy int

// ExtendStrategy enum
const (
	ExtendBlack      ExtendStrategy = C.VIPS_EXTEND_BLACK
	ExtendCopy       ExtendStrategy = C.VIPS_EXTEND_COPY
	ExtendRepeat     ExtendStrategy = C.VIPS_EXTEND_REPEAT
	ExtendMirror     ExtendStrategy = C.VIPS_EXTEND_MIRROR
	ExtendWhite      ExtendStrategy = C.VIPS_EXTEND_WHITE
	ExtendBackground ExtendStrategy = C.VIPS_EXTEND_BACKGROUND
)

// Interesting represents VIPS_INTERESTING type
// https://libvips.github.io/libvips/API/current/libvips-conversion.html#VipsInteresting
type Interesting int

// Interesting constants represent areas of interest which smart cropping will crop based on.
const (
	InterestingNone      Interesting = C.VIPS_INTERESTING_NONE
	InterestingCentre    Interesting = C.VIPS_INTERESTING_CENTRE
	InterestingEntropy   Interesting = C.VIPS_INTERESTING_ENTROPY
	InterestingAttention Interesting = C.VIPS_INTERESTING_ATTENTION
	InterestingLow       Interesting = C.VIPS_INTERESTING_LOW
	InterestingHigh      Interesting = C.VIPS_INTERESTING_HIGH
	InterestingAll       Interesting = C.VIPS_INTERESTING_ALL
	InterestingLast      Interesting = C.VIPS_INTERESTING_LAST
)

// SubsampleMode correlates to a libvips subsample mode
type SubsampleMode int

// SubsampleMode enum correlating to libvips subsample modes
const (
	VipsForeignSubsampleAuto SubsampleMode = C.VIPS_FOREIGN_JPEG_SUBSAMPLE_AUTO
	VipsForeignSubsampleOn   SubsampleMode = C.VIPS_FOREIGN_JPEG_SUBSAMPLE_ON
	VipsForeignSubsampleOff  SubsampleMode = C.VIPS_FOREIGN_JPEG_SUBSAMPLE_OFF
	VipsForeignSubsampleLast SubsampleMode = C.VIPS_FOREIGN_JPEG_SUBSAMPLE_LAST
)

// TiffCompression represents method for compressing a tiff at export
type TiffCompression int

// TiffCompression enum
const (
	TiffCompressionNone     TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_NONE
	TiffCompressionJpeg     TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_JPEG
	TiffCompressionDeflate  TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_DEFLATE
	TiffCompressionPackbits TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_PACKBITS
	TiffCompressionFax4     TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_CCITTFAX4
	TiffCompressionLzw      TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_LZW
	TiffCompressionWebp     TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_WEBP
	TiffCompressionZstd     TiffCompression = C.VIPS_FOREIGN_TIFF_COMPRESSION_ZSTD
)

// TiffPredictor represents method for compressing a tiff at export
type TiffPredictor int

// TiffPredictor enum
const (
	TiffPredictorNone       TiffPredictor = C.VIPS_FOREIGN_TIFF_PREDICTOR_NONE
	TiffPredictorHorizontal TiffPredictor = C.VIPS_FOREIGN_TIFF_PREDICTOR_HORIZONTAL
	TiffPredictorFloat      TiffPredictor = C.VIPS_FOREIGN_TIFF_PREDICTOR_FLOAT
)

// PngFilter represents filter algorithms that can be applied before compression.
// See https://www.w3.org/TR/PNG-Filters.html
type PngFilter int

// PngFilter enum
const (
	PngFilterNone  PngFilter = C.VIPS_FOREIGN_PNG_FILTER_NONE
	PngFilterSub   PngFilter = C.VIPS_FOREIGN_PNG_FILTER_SUB
	PngFilterUo    PngFilter = C.VIPS_FOREIGN_PNG_FILTER_UP
	PngFilterAvg   PngFilter = C.VIPS_FOREIGN_PNG_FILTER_AVG
	PngFilterPaeth PngFilter = C.VIPS_FOREIGN_PNG_FILTER_PAETH
	PngFilterAll   PngFilter = C.VIPS_FOREIGN_PNG_FILTER_ALL
)

// Size represents VipsSize type
type Size int

const (
	SizeBoth  Size = C.VIPS_SIZE_BOTH
	SizeUp    Size = C.VIPS_SIZE_UP
	SizeDown  Size = C.VIPS_SIZE_DOWN
	SizeForce Size = C.VIPS_SIZE_FORCE
	SizeLast  Size = C.VIPS_SIZE_LAST
)
