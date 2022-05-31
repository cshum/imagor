package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Version = "0.8.24"

// Loader load image from source
type Loader interface {
	Get(r *http.Request, image string) (*Bytes, error)
}

// Storage load and save image
type Storage interface {
	Get(r *http.Request, image string) (*Bytes, error)
	Put(ctx context.Context, image string, blob *Bytes) error
	Stat(ctx context.Context, image string) (*Stat, error)
}

// LoadFunc load function for Processor
type LoadFunc func(string) (*Bytes, error)

// Processor process image buffer
type Processor interface {
	Startup(ctx context.Context) error
	Process(ctx context.Context, blob *Bytes, p imagorpath.Params, load LoadFunc) (*Bytes, error)
	Shutdown(ctx context.Context) error
}

// ResultKey generator
type ResultKey interface {
	Generate(p imagorpath.Params) string
}

// Imagor image resize HTTP handler
type Imagor struct {
	Unsafe             bool
	Signer             imagorpath.Signer
	BasePathRedirect   string
	Loaders            []Loader
	Storages           []Storage
	ResultLoaders      []Loader
	ResultStorages     []Storage
	Processors         []Processor
	RequestTimeout     time.Duration
	LoadTimeout        time.Duration
	SaveTimeout        time.Duration
	ProcessTimeout     time.Duration
	CacheHeaderTTL     time.Duration
	ProcessConcurrency int64
	AutoWebP           bool
	AutoAVIF           bool
	ModifiedTimeCheck  bool
	Logger             *zap.Logger
	Debug              bool
	ResultKey          ResultKey

	g    singleflight.Group
	sema *semaphore.Weighted
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
	if app.ProcessConcurrency > 0 {
		app.sema = semaphore.NewWeighted(app.ProcessConcurrency)
	}
	if app.Debug {
		app.debugLog()
	}
	if app.Signer == nil {
		app.Signer = imagorpath.NewDefaultSigner("")
	}
	app.ResultLoaders = append(loaderSlice(app.ResultStorages), app.ResultLoaders...)
	app.Loaders = append(loaderSlice(app.Storages), app.Loaders...)
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.EscapedPath()
	if path == "/" || path == "" {
		if app.BasePathRedirect == "" {
			resJSON(w, json.RawMessage(fmt.Sprintf(
				`{"imagor":{"version":"%s"}}`, Version,
			)))
		} else {
			http.Redirect(w, r, app.BasePathRedirect, http.StatusTemporaryRedirect)
		}
		return
	}
	p := imagorpath.Parse(path)
	if p.Params {
		resJSONIndent(w, p)
		return
	}
	blob, err := app.Do(r, p)
	var buf []byte
	var ln int
	if !isEmpty(blob) {
		buf, _ = blob.ReadAll()
		ln = len(buf)
		if blob.Meta != nil && p.Meta {
			resJSON(w, blob.Meta)
			return
		}
		if ln > 0 {
			w.Header().Set("Content-Type", blob.ContentType())
		}
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
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
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf)
	}
	return
}

// Do executes Imagor operations
func (app *Imagor) Do(r *http.Request, p imagorpath.Params) (blob *Bytes, err error) {
	var cancel func()
	ctx := r.Context()
	if app.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, app.RequestTimeout)
		defer cancel()
		r = r.WithContext(ctx)
	}
	if !(app.Unsafe && p.Unsafe) && app.Signer != nil && app.Signer.Sign(p.Path) != p.Hash {
		err = ErrSignatureMismatch
		if app.Debug {
			app.Logger.Debug("sign-mismatch", zap.Any("params", p), zap.String("expected", app.Signer.Sign(p.Path)))
		}
		return
	}
	// auto WebP / AVIF
	if app.AutoWebP || app.AutoAVIF {
		var hasFormat bool
		for _, f := range p.Filters {
			if f.Name == "format" {
				hasFormat = true
			}
		}
		if !hasFormat {
			accept := r.Header.Get("Accept")
			if strings.Contains(accept, "image/avif") && app.AutoAVIF {
				p.Filters = append(p.Filters, imagorpath.Filter{
					Name: "format",
					Args: "avif",
				})
				p.Path = imagorpath.GeneratePath(p)
			} else if strings.Contains(accept, "image/webp") && app.AutoWebP {
				p.Filters = append(p.Filters, imagorpath.Filter{
					Name: "format",
					Args: "webp",
				})
				p.Path = imagorpath.GeneratePath(p)
			}
		}
	}
	var resultKey string
	if app.ResultKey != nil {
		resultKey = app.ResultKey.Generate(p)
	} else {
		resultKey = strings.TrimPrefix(p.Path, "meta/")
	}
	load := func(image string) (*Bytes, error) {
		return app.loadStorage(r, image)
	}
	return app.suppress(ctx, "res:"+resultKey, func(ctx context.Context) (*Bytes, error) {
		if blob, resOrigin, err := app.load(
			r, app.ResultLoaders, resultKey,
		); err == nil && !isEmpty(blob) {
			if app.ModifiedTimeCheck && resOrigin != nil {
				if resStat, err1 := resOrigin.Stat(ctx, resultKey); resStat != nil && err1 == nil {
					if sourceStat, err2 := app.storageStat(ctx, p.Image); sourceStat != nil && err2 == nil {
						if !resStat.ModifiedTime.Before(sourceStat.ModifiedTime) {
							return blob, nil
						}
					}
				}
			} else {
				return blob, nil
			}
		}
		if app.sema != nil {
			if err = app.sema.Acquire(ctx, 1); err != nil {
				app.Logger.Debug("acquire", zap.Error(err))
				return blob, err
			}
			defer app.sema.Release(1)
		}
		if blob, err = app.loadStorage(r, p.Image); err != nil {
			app.Logger.Debug("load", zap.Any("params", p), zap.Error(err))
			return blob, err
		}
		if isEmpty(blob) {
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
					if !isEmpty(f) {
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

func (app *Imagor) loadStorage(r *http.Request, key string) (*Bytes, error) {
	return app.suppress(r.Context(), "img:"+key, func(ctx context.Context) (blob *Bytes, err error) {
		var origin Storage
		r = r.WithContext(ctx)
		blob, origin, err = app.load(r, app.Loaders, key)
		if err != nil || isEmpty(blob) {
			return
		}
		if len(app.Storages) > 0 {
			app.save(ctx, origin, app.Storages, key, blob)
		}
		return
	})
}

func (app *Imagor) load(
	r *http.Request, loaders []Loader, key string,
) (blob *Bytes, origin Storage, err error) {
	if len(loaders) == 0 {
		return
	}
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
		f, e := loader.Get(loadReq, key)
		if !isEmpty(f) {
			blob = f
		}
		if e == nil {
			err = nil
			origin, _ = loader.(Storage)
			break
		}
		// should not log expected error as of now, as it has not reached the end
		err = e
	}
	if err == ErrPass {
		// pass till the end means not found
		err = ErrNotFound
	}
	if app.Debug {
		if err == nil {
			app.Logger.Debug("loaded", zap.String("key", key))
		} else {
			app.Logger.Debug("load", zap.String("key", key), zap.Error(err))
		}
	}
	return
}

func (app *Imagor) storageStat(ctx context.Context, key string) (stat *Stat, err error) {
	if len(app.Storages) == 0 {
		return
	}
	for _, storage := range app.Storages {
		if stat, err = storage.Stat(ctx, key); stat != nil && err == nil {
			return
		}
	}
	return
}

func (app *Imagor) save(
	ctx context.Context, origin Storage, storages []Storage, key string, blob *Bytes,
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
			if err := storage.Put(ctx, key, blob); err != nil {
				app.Logger.Warn("save", zap.String("key", key), zap.Error(err))
			} else if app.Debug {
				app.Logger.Debug("saved", zap.String("key", key))
			}
		}(storage)
	}
	wg.Wait()
	return
}

type suppressKey struct {
	Key string
}

func (app *Imagor) suppress(
	ctx context.Context,
	key string, fn func(ctx context.Context) (*Bytes, error),
) (blob *Bytes, err error) {
	if app.Debug {
		app.Logger.Debug("suppress", zap.String("key", key))
	}
	if isAcquired, ok := ctx.Value(suppressKey{key}).(bool); ok && isAcquired {
		// resolve deadlock
		return fn(ctx)
	}
	isCanceled := false
	ch := app.g.DoChan(key, func() (interface{}, error) {
		v, err := fn(context.WithValue(ctx, suppressKey{key}, true))
		if errors.Is(err, context.Canceled) {
			app.g.Forget(key)
			isCanceled = true
		}
		return v, err
	})
	select {
	case res := <-ch:
		if !isCanceled && errors.Is(res.Err, context.Canceled) {
			// resolve canceled
			return app.suppress(ctx, key, fn)
		}
		if res.Val != nil {
			return res.Val.(*Bytes), res.Err
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
	var loaders, storages, resultStorages, processors []string
	for _, v := range app.Loaders {
		loaders = append(loaders, getType(v))
	}
	for _, v := range app.Storages {
		storages = append(storages, getType(v))
	}
	for _, v := range app.Processors {
		processors = append(processors, getType(v))
	}
	for _, v := range app.ResultStorages {
		resultStorages = append(resultStorages, getType(v))
	}
	app.Logger.Debug("imagor",
		zap.String("version", Version),
		zap.Bool("unsafe", app.Unsafe),
		zap.Duration("request_timeout", app.RequestTimeout),
		zap.Duration("load_timeout", app.LoadTimeout),
		zap.Duration("process_timeout", app.ProcessTimeout),
		zap.Duration("save_timeout", app.SaveTimeout),
		zap.Int64("process_concurrency", app.ProcessConcurrency),
		zap.Duration("cache_header_ttl", app.CacheHeaderTTL),
		zap.Strings("loaders", loaders),
		zap.Strings("storages", storages),
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

func loaderSlice(storages []Storage) (loaders []Loader) {
	for _, storage := range storages {
		loaders = append(loaders, storage)
	}
	return
}
