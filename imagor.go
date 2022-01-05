package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Version = "0.6.1"

// Loader Load image from image source
type Loader interface {
	Load(r *http.Request, image string) (*Blob, error)
}

// LoadFunc imagor load function for Processor
type LoadFunc func(string) (*Blob, error)

// Storage save image buffer
type Storage interface {
	Save(ctx context.Context, image string, blob *Blob) error
}

// Store both a Loader and Storage
type Store interface {
	Loader
	Storage
}

// Processor process image buffer
type Processor interface {
	Startup(ctx context.Context) error
	Process(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error)
	Shutdown(ctx context.Context) error
}

// Imagor image resize HTTP handler
type Imagor struct {
	Unsafe         bool
	Secret         string
	Loaders        []Loader
	Storages       []Storage
	ResultLoaders  []Loader
	ResultStorages []Storage
	Processors     []Processor
	RequestTimeout time.Duration
	LoadTimeout    time.Duration
	SaveTimeout    time.Duration
	ProcessTimeout time.Duration
	CacheHeaderTTL time.Duration
	Logger         *zap.Logger
	Debug          bool

	g singleflight.Group
}

// New create new Imagor
func New(options ...Option) *Imagor {
	app := &Imagor{
		Logger:         zap.NewNop(),
		RequestTimeout: time.Second * 30,
		LoadTimeout:    time.Second * 20,
		SaveTimeout:    time.Second * 20,
		ProcessTimeout: time.Second * 20,
		CacheHeaderTTL: time.Hour * 24,
	}
	for _, option := range options {
		option(app)
	}
	if app.Debug {
		app.debugLog()
	}
	return app
}

// Startup Imagor startup lifecycle
func (app *Imagor) Startup(ctx context.Context) (err error) {
	for _, processor := range app.Processors {
		if err = processor.Startup(ctx); err != nil {
			return
		}
	}
	return
}

// Shutdown Imagor shutdown lifecycle
func (app *Imagor) Shutdown(ctx context.Context) (err error) {
	for _, processor := range app.Processors {
		if err = processor.Shutdown(ctx); err != nil {
			return
		}
	}
	return
}

// ServeHTTP implements http.Handler for Imagor operations
func (app *Imagor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.EscapedPath()
	if path == "/" || path == "" {
		resJSON(w, json.RawMessage(fmt.Sprintf(
			`{"imagor":{"version":"%s"}}`, Version,
		)))
		return
	}
	p := imagorpath.Parse(path)
	if p.Params {
		resJSONIndent(w, p)
		return
	}
	file, err := app.Do(r, p)
	var buf []byte
	var ln int
	if !IsFileEmpty(file) {
		buf, _ = file.ReadAll()
		ln = len(buf)
		if file.Meta != nil {
			if p.Meta {
				resJSON(w, file.Meta)
				return
			} else {
				w.Header().Set("Content-Type", file.Meta.ContentType)
			}
		} else if ln > 0 {
			w.Header().Set("Content-Type", http.DetectContentType(buf))
		}
	}
	if err != nil {
		if e, ok := WrapError(err).(Error); ok {
			if e == ErrPass {
				// passed till the end means not found
				e = ErrNotFound
			}
			w.WriteHeader(e.Code)
			if ln > 0 {
				w.Header().Set("Content-Length", strconv.Itoa(ln))
				_, _ = w.Write(buf)
				return
			}
			resJSON(w, e)
		} else {
			resJSON(w, ErrInternal)
		}
		return
	}
	setCacheHeaders(w, app.CacheHeaderTTL)
	w.Header().Set("Content-Length", strconv.Itoa(ln))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf)
	return
}

// Do executes Imagor operations
func (app *Imagor) Do(r *http.Request, p imagorpath.Params) (blob *Blob, err error) {
	var cancel func()
	ctx := r.Context()
	if app.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, app.RequestTimeout)
		defer cancel()
		r = r.WithContext(ctx)
	}
	if !(app.Unsafe && p.Unsafe) && imagorpath.Sign(p.Path, app.Secret) != p.Hash {
		err = ErrSignatureMismatch
		if app.Debug {
			app.Logger.Debug("sign-mismatch", zap.Any("params", p), zap.String("expected", imagorpath.Sign(p.Path, app.Secret)))
		}
		return
	}
	resultKey := strings.TrimPrefix(p.Path, "meta/")
	load := func(image string) (*Blob, error) {
		return app.loadStore(r, image)
	}
	return app.acquire(ctx, "res:"+resultKey, func(ctx context.Context) (*Blob, error) {
		if blob, err = app.loadResult(r, resultKey); err == nil && !IsFileEmpty(blob) {
			return blob, err
		}
		if blob, err = app.loadStore(r, p.Image); err != nil {
			app.Logger.Debug("load", zap.Any("params", p), zap.Error(err))
			return blob, err
		}
		if IsFileEmpty(blob) {
			return blob, err
		}
		var cancel func()
		if app.ProcessTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, app.ProcessTimeout)
			defer cancel()
		}
		for _, processor := range app.Processors {
			f, e := processor.Process(ctx, blob, p, load)
			if e == nil {
				blob = f
				err = nil
				if app.Debug {
					app.Logger.Debug("processed", zap.Any("params", p), zap.Any("meta", f.Meta))
				}
				break
			} else {
				if e == ErrPass {
					if !IsFileEmpty(f) {
						// pass to next processor
						blob = f
					}
					if app.Debug {
						app.Logger.Debug("process", zap.Any("params", p), zap.Error(e))
					}
				} else {
					err = e
					app.Logger.Warn("process", zap.Any("params", p), zap.Error(err))
					if errors.Is(err, context.DeadlineExceeded) {
						break
					}
				}
			}
		}
		if err == nil && len(app.ResultStorages) > 0 {
			app.save(ctx, nil, app.ResultStorages, resultKey, blob)
		}
		return blob, err
	})
}

func (app *Imagor) loadStore(r *http.Request, key string) (*Blob, error) {
	return app.acquire(r.Context(), "img:"+key, func(ctx context.Context) (blob *Blob, err error) {
		var origin Store
		r = r.WithContext(ctx)
		blob, origin, err = app.load(r, app.Loaders, key)
		if IsFileEmpty(blob) {
			return
		}
		if len(app.Storages) > 0 {
			app.save(ctx, origin, app.Storages, key, blob)
		}
		return
	})
}

func (app *Imagor) loadResult(r *http.Request, key string) (blob *Blob, err error) {
	if len(app.ResultLoaders) == 0 {
		return
	}
	blob, _, err = app.load(r, app.ResultLoaders, key)
	return
}

func (app *Imagor) load(
	r *http.Request, loaders []Loader, key string,
) (blob *Blob, origin Store, err error) {
	var ctx = r.Context()
	var loadCtx = ctx
	var loadReq = r
	var cancel func()
	if app.LoadTimeout > 0 {
		loadCtx, cancel = context.WithTimeout(loadCtx, app.LoadTimeout)
		defer cancel()
		loadReq = r.WithContext(loadCtx)
	}
	for _, loader := range loaders {
		f, e := loader.Load(loadReq, key)
		if !IsFileEmpty(f) {
			blob = f
		}
		if e == nil {
			err = nil
			origin, _ = loader.(Store)
			break
		}
		// should not log expected error as of now, as it has not reached the end
		if e != nil {
			if app.Debug || (e != ErrPass && e != ErrNotFound && !errors.Is(e, context.Canceled)) {
				app.Logger.Warn("load", zap.String("key", key), zap.Error(e))
			}
		}
		err = e
	}
	if err == nil {
		if app.Debug {
			app.Logger.Debug("loaded", zap.String("key", key))
		}
	} else if !errors.Is(err, context.Canceled) {
		if err == ErrPass {
			err = ErrNotFound
		}
		// log non user-initiated error finally
		app.Logger.Warn("load", zap.String("key", key), zap.Error(err))
	}
	return
}

func (app *Imagor) save(
	ctx context.Context, origin Storage, storages []Storage, key string, blob *Blob,
) {
	var cancel func()
	if app.SaveTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, app.SaveTimeout)
	}
	defer cancel()
	var wg sync.WaitGroup
	for _, storage := range storages {
		if storage == origin {
			// loaded from the same store, no need save again
			if app.Debug {
				app.Logger.Debug("skip-save", zap.String("key", key))
			}
			continue
		}
		wg.Add(1)
		go func(storage Storage) {
			defer wg.Done()
			if err := storage.Save(ctx, key, blob); err != nil {
				app.Logger.Warn("save", zap.String("key", key), zap.Error(err))
			} else if app.Debug {
				app.Logger.Debug("saved", zap.String("key", key))
			}
		}(storage)
	}
	wg.Wait()
	return
}

type acquireKey struct {
	Key string
}

func (app *Imagor) acquire(
	ctx context.Context,
	key string, fn func(ctx context.Context) (*Blob, error),
) (blob *Blob, err error) {
	if app.Debug {
		app.Logger.Debug("acquire", zap.String("key", key))
	}
	if isAcquired, ok := ctx.Value(acquireKey{key}).(bool); ok && isAcquired {
		// resolve deadlock
		return fn(ctx)
	}
	ch := app.g.DoChan(key, func() (interface{}, error) {
		ctx = context.WithValue(ctx, acquireKey{key}, true)
		return fn(ctx)
	})
	select {
	case res := <-ch:
		if res.Val != nil {
			return res.Val.(*Blob), res.Err
		}
		return nil, res.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (app *Imagor) debugLog() {
	if !app.Debug {
		return
	}
	var loaders, storages, resultLoaders, resultStorages, processors []string
	for _, v := range app.Loaders {
		loaders = append(loaders, getType(v))
	}
	for _, v := range app.Storages {
		storages = append(storages, getType(v))
	}
	for _, v := range app.Processors {
		processors = append(processors, getType(v))
	}
	for _, v := range app.ResultLoaders {
		resultLoaders = append(resultLoaders, getType(v))
	}
	for _, v := range app.ResultStorages {
		resultStorages = append(resultStorages, getType(v))
	}
	app.Logger.Debug("imagor",
		zap.Bool("unsafe", app.Unsafe),
		zap.Duration("request_timeout", app.RequestTimeout),
		zap.Duration("load_timeout", app.LoadTimeout),
		zap.Duration("save_timeout", app.SaveTimeout),
		zap.Duration("cache_header_ttl", app.CacheHeaderTTL),
		zap.Strings("loaders", loaders),
		zap.Strings("storages", storages),
		zap.Strings("result_loaders", resultLoaders),
		zap.Strings("result_storages", resultStorages),
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
	_, _ = w.Write(buf)
	return
}

func resJSONIndent(w http.ResponseWriter, v interface{}) {
	buf, _ := json.MarshalIndent(v, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	_, _ = w.Write(buf)
	return
}

func getType(v interface{}) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}
