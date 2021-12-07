package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshum/hybridcache"
	"go.uber.org/zap"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type LoadFunc func(string) ([]byte, error)

// Loader Load image from image source
type Loader interface {
	Load(r *http.Request, image string) ([]byte, error)
}

// Storage save image buffer
type Storage interface {
	Save(ctx context.Context, image string, buf []byte) error
}

// Store both a Loader and Storage
type Store interface {
	Loader
	Storage
}

// Processor process image buffer
type Processor interface {
	Startup(ctx context.Context) error
	Process(ctx context.Context, buf []byte, params Params, load LoadFunc) ([]byte, *Meta, error)
	Shutdown(ctx context.Context) error
}

// Imagor image resize HTTP handler
type Imagor struct {
	Version        string
	Unsafe         bool
	Secret         string
	Loaders        []Loader
	Storages       []Storage
	Processors     []Processor
	RequestTimeout time.Duration
	LoadTimeout    time.Duration
	SaveTimeout    time.Duration
	Cache          cache.Cache
	CacheHeaderTTL time.Duration
	CacheTTL       time.Duration
	Logger         *zap.Logger
	Debug          bool
}

// New create new Imagor
func New(options ...Option) *Imagor {
	app := &Imagor{
		Version:        "dev",
		Logger:         zap.NewNop(),
		RequestTimeout: time.Second * 30,
		LoadTimeout:    time.Second * 20,
		SaveTimeout:    time.Second * 20,
		CacheHeaderTTL: time.Hour * 24,
		CacheTTL:       time.Minute,
	}
	for _, option := range options {
		option(app)
	}
	app.Cache = cache.NewMemory(1000, 1<<28, app.CacheTTL)
	if app.LoadTimeout > app.RequestTimeout || app.LoadTimeout == 0 {
		app.LoadTimeout = app.RequestTimeout
	}
	if app.Debug {
		app.debugLog()
	}
	return app
}

func (app *Imagor) Startup(ctx context.Context) (err error) {
	for _, processor := range app.Processors {
		if err = processor.Startup(ctx); err != nil {
			return
		}
	}
	return
}

func (app *Imagor) Shutdown(ctx context.Context) (err error) {
	for _, processor := range app.Processors {
		if err = processor.Shutdown(ctx); err != nil {
			return
		}
	}
	return
}

func (app *Imagor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.EscapedPath()
	if uri == "/" {
		resJSON(w, json.RawMessage(fmt.Sprintf(
			`{"imagor":{"version":"%s"}}`, app.Version,
		)))
		return
	}
	params := ParseParams(uri)
	if params.Params {
		setCacheHeaders(w, 0)
		resJSONIndent(w, params)
		return
	}
	buf, meta, err := app.Do(r, params)
	ln := len(buf)
	if meta != nil {
		if params.Meta {
			setCacheHeaders(w, 0)
			resJSON(w, meta)
			return
		} else {
			w.Header().Set("Content-Type", meta.ContentType)
		}
	} else if ln > 0 {
		w.Header().Set("Content-Type", http.DetectContentType(buf))
	}
	if err != nil {
		if e, ok := WrapError(err).(Error); ok {
			if e == ErrPass {
				// passed till the end means not found
				e = ErrNotFound
			}
			w.WriteHeader(e.Code)
		}
		if ln > 0 {
			w.Header().Set("Content-Length", strconv.Itoa(ln))
			w.Write(buf)
			return
		}
		resJSON(w, err)
		return
	}
	setCacheHeaders(w, app.CacheHeaderTTL)
	w.Header().Set("Content-Length", strconv.Itoa(ln))
	w.Write(buf)
	return
}

func (app *Imagor) Do(r *http.Request, params Params) (buf []byte, meta *Meta, err error) {
	var cancel func()
	ctx := r.Context()
	if app.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, app.RequestTimeout)
		defer cancel()
	}
	if !(app.Unsafe && params.Unsafe) && !params.Verify(app.Secret) {
		err = ErrSignatureMismatch
		if app.Debug {
			app.Logger.Debug("sign-mismatch", zap.Any("params", params), zap.String("expected", Sign(params.Path, app.Secret)))
		}
		return
	}
	if buf, err = app.load(r, params.Image); err != nil {
		app.Logger.Debug("load", zap.Any("params", params), zap.Error(err))
		return
	}
	load := func(image string) ([]byte, error) {
		return app.load(r, image)
	}
	for _, processor := range app.Processors {
		b, m, e := processor.Process(ctx, buf, params, load)
		if e == nil {
			buf = b
			meta = m
			if app.Debug {
				app.Logger.Debug("processed", zap.Any("params", params), zap.Any("meta", meta), zap.Int("size", len(buf)))
			}
			break
		} else {
			if e == ErrPass {
				if len(b) > 0 {
					// pass to next processor
					buf = b
				}
				if app.Debug {
					app.Logger.Debug("process", zap.Any("params", params), zap.Error(e))
				}
			} else {
				err = e
				app.Logger.Warn("process", zap.Any("params", params), zap.Error(e))
			}
		}
	}
	return
}

func (app *Imagor) load(r *http.Request, image string) (buf []byte, err error) {
	buf, err = cache.NewFunc(
		app.Cache, app.LoadTimeout, app.CacheTTL, app.CacheTTL,
	).DoBytes(r.Context(), image, func(ctx context.Context) (buf []byte, err error) {
		dr := r.WithContext(ctx)
		for _, loader := range app.Loaders {
			b, e := loader.Load(dr, image)
			if len(b) > 0 {
				buf = b
			}
			if e == nil {
				err = nil
				break
			}
			// should not log expected error as of now, as it has not reached the end
			if e != nil && e != ErrPass && e != ErrNotFound && !errors.Is(e, context.Canceled) {
				app.Logger.Warn("load", zap.String("image", image), zap.Error(e))
			} else if app.Debug {
				app.Logger.Debug("load", zap.String("image", image), zap.Error(e))
			}
			err = e
		}
		if err == nil {
			if app.Debug {
				app.Logger.Debug("loaded", zap.String("image", image), zap.Int("size", len(buf)))
			}
			if len(app.Storages) > 0 {
				app.save(ctx, app.Storages, image, buf)
			}
		} else if !errors.Is(err, context.Canceled) {
			if err == ErrPass {
				err = ErrNotFound
			}
			// log non user-initiated error finally
			app.Logger.Warn("load", zap.String("image", image), zap.Error(err))
		}
		return
	})
	// wrap error to handle cache serialization
	err = WrapError(err)
	return
}

func (app *Imagor) save(
	ctx context.Context, storages []Storage, image string, buf []byte,
) {
	for _, storage := range storages {
		var cancel func()
		sCtx := DetachContext(ctx)
		if app.SaveTimeout > 0 {
			sCtx, cancel = context.WithTimeout(sCtx, app.SaveTimeout)
		}
		go func(s Storage) {
			defer cancel()
			if err := s.Save(sCtx, image, buf); err != nil {
				app.Logger.Warn("save", zap.String("image", image), zap.Error(err))
			} else if app.Debug {
				app.Logger.Debug("saved", zap.String("image", image), zap.Int("size", len(buf)))
			}
		}(storage)
	}
}

func (app *Imagor) debugLog() {
	if !app.Debug {
		return
	}
	var loaders, storages, processors []string
	for _, v := range app.Loaders {
		loaders = append(loaders, getType(v))
	}
	for _, v := range app.Storages {
		storages = append(storages, getType(v))
	}
	for _, v := range app.Processors {
		processors = append(processors, getType(v))
	}
	app.Logger.Debug("imagor",
		zap.Bool("unsafe", app.Unsafe),
		zap.Duration("request_timeout", app.RequestTimeout),
		zap.Duration("load_timeout", app.LoadTimeout),
		zap.Duration("save_timeout", app.SaveTimeout),
		zap.Duration("cache_header_ttl", app.CacheHeaderTTL),
		zap.Strings("loaders", loaders),
		zap.Strings("storages", storages),
		zap.Strings("processors", processors),
	)
}

func setCacheHeaders(w http.ResponseWriter, ttl time.Duration) {
	expires := time.Now().Add(ttl)

	w.Header().Add("Expires", strings.Replace(expires.Format(time.RFC1123), "UTC", "GMT", -1))
	w.Header().Add("Cache-Control", getCacheControl(ttl))
}

func getCacheControl(ttl time.Duration) string {
	if ttl == 0 {
		return "private, no-cache, no-store, must-revalidate"
	}
	ttlSec := int(ttl.Seconds())
	return fmt.Sprintf("public, s-maxage=%d, max-age=%d, no-transform", ttlSec, ttlSec)
}

func resJSON(w http.ResponseWriter, v interface{}) {
	buf, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.Write(buf)
	return
}

func resJSONIndent(w http.ResponseWriter, v interface{}) {
	buf, _ := json.MarshalIndent(v, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.Write(buf)
	return
}

func getType(v interface{}) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}
