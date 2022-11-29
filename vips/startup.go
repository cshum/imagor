package vips

// #cgo pkg-config: vips
// #include <vips/vips.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"runtime"
	"sync"
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
	lock                    sync.Mutex
	once                    sync.Once
	isStarted               bool
	isShutdown              bool
	supportedLoadImageTypes = make(map[ImageType]bool)
	supportedSaveImageTypes = make(map[ImageType]bool)
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
	lock.Lock()
	defer lock.Unlock()

	if isStarted || isShutdown {
		return
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if C.VIPS_MAJOR_VERSION < 8 {
		panic("vips requires libvips version 8.10+")
	}

	if C.VIPS_MAJOR_VERSION == 8 && C.VIPS_MINOR_VERSION < 10 {
		panic("vips requires libvips version 8.10+")
	}

	cName := C.CString("vips")
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

	log("vips", LogLevelInfo, fmt.Sprintf("vips %s started with concurrency=%d cache_max_files=%d cache_max_mem=%d cache_max=%d",
		Version,
		int(C.vips_concurrency_get()),
		int(C.vips_cache_get_max_files()),
		int(C.vips_cache_get_max_mem()),
		int(C.vips_cache_get_max())))

	cType := C.CString("VipsOperation")
	defer freeCString(cType)

	for k, v := range ImageTypes {
		func() {
			cLoad := C.CString(v + "load")
			defer freeCString(cLoad)

			supportLoad := C.vips_type_find(cType, cLoad)
			supportedLoadImageTypes[k] = int(supportLoad) != 0

			cSave := C.CString(v + "save_buffer")
			defer freeCString(cSave)
			supportSave := C.vips_type_find(cType, cSave)
			supportedSaveImageTypes[k] = int(supportSave) != 0
		}()
		if supportedLoadImageTypes[k] || supportedSaveImageTypes[k] {
			log("vips", LogLevelInfo, fmt.Sprintf(
				"registered image type=%s load=%t save=%t",
				v, IsLoadSupported(k), IsSaveSupported(k)))
		}
	}
	isStarted = true
}

func startupIfNeeded() {
	once.Do(func() {
		Startup(nil)
	})
}

// Shutdown libvips
func Shutdown() {
	lock.Lock()
	defer lock.Unlock()

	if !isStarted || isShutdown {
		return
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	C.vips_shutdown()
	disableLogging()

	isShutdown = true
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
