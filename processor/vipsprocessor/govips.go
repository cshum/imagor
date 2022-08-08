package vipsprocessor

// #cgo pkg-config: vips
// #include <vips/vips.h>
// #include <stdlib.h>
// #include <glib.h>
// #include "govips.h"
import "C"
import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// Version is the full libvips version string (x.y.z)
const Version = string(C.VIPS_VERSION)

// MajorVersion is the libvips major component of the version string (x in x.y.z)
const MajorVersion = int(C.VIPS_MAJOR_VERSION)

// MinorVersion is the libvips minor component of the version string (y in x.y.z)
const MinorVersion = int(C.VIPS_MINOR_VERSION)

// MicroVersion is the libvips micro component of the version string (z in x.y.z)
// Also known as patch version
const MicroVersion = int(C.VIPS_MICRO_VERSION)

const (
	defaultConcurrencyLevel = 1
	defaultMaxCacheMem      = 50 * 1024 * 1024
	defaultMaxCacheSize     = 100
	defaultMaxCacheFiles    = 0
)

var (
	initLock            sync.Mutex
	supportedImageTypes = make(map[ImageType]bool)
)

type config struct {
	ConcurrencyLevel int
	MaxCacheFiles    int
	MaxCacheMem      int
	MaxCacheSize     int
	ReportLeaks      bool
	CacheTrace       bool
}

// Startup sets up the libvips support and ensures the versions are correct. Pass in nil for
// default configuration.
func Startup(config *config) {
	initLock.Lock()
	defer initLock.Unlock()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if C.VIPS_MAJOR_VERSION < 8 {
		panic("govips requires libvips version 8.10+")
	}

	if C.VIPS_MAJOR_VERSION == 8 && C.VIPS_MINOR_VERSION < 10 {
		panic("govips requires libvips version 8.10+")
	}

	cName := C.CString("govips")
	defer freeCString(cName)

	// Override default glib logging handler to intercept logging messages
	enableLogging()

	err := C.vips_init(cName)
	if err != 0 {
		panic(fmt.Sprintf("Failed to start vips code=%v", err))
	}

	if config != nil {

		C.vips_leak_set(toGboolean(config.ReportLeaks))

		if config.ConcurrencyLevel >= 0 {
			C.vips_concurrency_set(C.int(config.ConcurrencyLevel))
		} else {
			C.vips_concurrency_set(defaultConcurrencyLevel)
		}

		if config.MaxCacheFiles >= 0 {
			C.vips_cache_set_max_files(C.int(config.MaxCacheFiles))
		} else {
			C.vips_cache_set_max_files(defaultMaxCacheFiles)
		}

		if config.MaxCacheMem >= 0 {
			C.vips_cache_set_max_mem(C.size_t(config.MaxCacheMem))
		} else {
			C.vips_cache_set_max_mem(defaultMaxCacheMem)
		}

		if config.MaxCacheSize >= 0 {
			C.vips_cache_set_max(C.int(config.MaxCacheSize))
		} else {
			C.vips_cache_set_max(defaultMaxCacheSize)
		}

		if config.CacheTrace {
			C.vips_cache_set_trace(toGboolean(true))
		}
	} else {
		C.vips_concurrency_set(defaultConcurrencyLevel)
		C.vips_cache_set_max(defaultMaxCacheSize)
		C.vips_cache_set_max_mem(defaultMaxCacheMem)
		C.vips_cache_set_max_files(defaultMaxCacheFiles)
	}

	log("govips", LogLevelInfo, fmt.Sprintf("vips %s started with concurrency=%d cache_max_files=%d cache_max_mem=%d cache_max=%d",
		Version,
		int(C.vips_concurrency_get()),
		int(C.vips_cache_get_max_files()),
		int(C.vips_cache_get_max_mem()),
		int(C.vips_cache_get_max())))

	cType := C.CString("VipsOperation")
	defer freeCString(cType)

	for k, v := range ImageTypes {
		cFunc := C.CString(v + "load")
		//noinspection GoDeferInLoop
		defer freeCString(cFunc)

		ret := C.vips_type_find(cType, cFunc)

		supportedImageTypes[k] = int(ret) != 0

		if supportedImageTypes[k] {
			log("govips", LogLevelInfo, fmt.Sprintf("registered image type loader type=%s", v))
		}
	}
}

func enableLogging() {
	C.vips_set_logging_handler()
}

func disableLogging() {
	C.vips_unset_logging_handler()
}

func Shutdown() {
	initLock.Lock()
	defer initLock.Unlock()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	C.vips_shutdown()
	disableLogging()
}

// MemoryStats is a data structure that houses various memory statistics from ReadVipsMemStats()
type MemoryStats struct {
	Mem     int64
	MemHigh int64
	Files   int64
	Allocs  int64
}

// ReadVipsMemStats returns various memory statistics such as allocated memory and open files.
func ReadVipsMemStats(stats *MemoryStats) {
	stats.Mem = int64(C.vips_tracked_get_mem())
	stats.MemHigh = int64(C.vips_tracked_get_mem_highwater())
	stats.Allocs = int64(C.vips_tracked_get_allocs())
	stats.Files = int64(C.vips_tracked_get_files())
}

type LogLevel int

const (
	LogLevelError    LogLevel = C.G_LOG_LEVEL_ERROR
	LogLevelCritical LogLevel = C.G_LOG_LEVEL_CRITICAL
	LogLevelWarning  LogLevel = C.G_LOG_LEVEL_WARNING
	LogLevelMessage  LogLevel = C.G_LOG_LEVEL_MESSAGE
	LogLevelInfo     LogLevel = C.G_LOG_LEVEL_INFO
	LogLevelDebug    LogLevel = C.G_LOG_LEVEL_DEBUG
)

var (
	currentLoggingHandlerFunction = noopLoggingHandler
	currentLoggingVerbosity       LogLevel
)

type LoggingHandlerFunction func(messageDomain string, messageLevel LogLevel, message string)

func loggingSettings(handler LoggingHandlerFunction, verbosity LogLevel) {
	if handler != nil {
		currentLoggingHandlerFunction = handler
	}
	currentLoggingVerbosity = verbosity
}

func noopLoggingHandler(_ string, _ LogLevel, _ string) {
}

func log(domain string, level LogLevel, message string) {
	if level <= currentLoggingVerbosity {
		currentLoggingHandlerFunction(domain, level, message)
	}
}

func handleImageError(out *C.VipsImage) error {
	if out != nil {
		clearImage(out)
	}
	return handleVipsError()
}

func handleVipsError() error {
	s := C.GoString(C.vips_error_buffer())
	C.vips_error_clear()

	return fmt.Errorf("%v", s)
}

func freeCString(s *C.char) {
	C.free(unsafe.Pointer(s))
}

func gFreePointer(ref unsafe.Pointer) {
	C.g_free(C.gpointer(ref))
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolToStr(v bool) string {
	if v {
		return "TRUE"
	}
	return "FALSE"
}

func toGboolean(b bool) C.gboolean {
	if b {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

func fromGboolean(b C.gboolean) bool {
	return b != 0
}

//func fromCArrayInt(out *C.int, n int) []int {
//	var result = make([]int, n)
//	var data []C.int
//	sh := (*reflect.SliceHeader)(unsafe.Pointer(&data))
//	sh.Data = uintptr(unsafe.Pointer(out))
//	sh.Len = n
//	sh.Cap = n
//	for i := range data {
//		result[i] = int(data[i])
//	}
//	return result
//}
