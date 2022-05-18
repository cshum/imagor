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

const Version = "0.8.16"

// Loader load image from source
type Loader interface {
	Load(r *http.Request, image string) (*Blob, error)
}

// Saver saves image
type Saver interface {
	Save(ctx context.Context, image string, blob *Blob) error
}

// Storage implements Loader and Saver
type Storage interface {
	Loader
	Saver
}

// LoadFunc imagor load function for Processor
type LoadFunc func(string) (*Blob, error)

// Processor process image buffer
type Processor interface {
	Startup(ctx context.Context) error
	Process(ctx context.Context, blob *Blob, p imagorpath.Params, load LoadFunc) (*Blob, error)
	Shutdown(ctx context.Context) error
}

// Imagor image resize HTTP handler
type Imagor struct {
	Unsafe             bool
	Secret             string
	BasePathRedirect   string
	Loaders            []Loader
	Savers             []Saver
	ResultLoaders      []Loader
	ResultSavers       []Saver
	Processors         []Processor
	RequestTimeout     time.Duration
	LoadTimeout        time.Duration
	SaveTimeout        time.Duration
	ProcessTimeout     time.Duration
	CacheHeaderTTL     time.Duration
	ProcessConcurrency int64
	AutoWebP           bool
	AutoAvif           bool
	Logger             *zap.Logger
	Debug              bool

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
	file, err := app.Do(r, p)
	var buf []byte
	var ln int
	if !IsBlobEmpty(file) {
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
			contentType := http.DetectContentType(buf)
			// mimesniff can't detect avif images
			if contentType == "application/octet-stream" {
				if file.IsAVIF() {
					contentType = "image/avif"
				}
			}
			w.Header().Set("Content-Type", contentType)
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
	// auto WebP / Avif
	if app.AutoWebP || app.AutoAvif {
		var hasFormat bool
		for _, f := range p.Filters {
			if f.Name == "format" {
				hasFormat = true
			}
		}
		if !hasFormat {
			accept := r.Header.Get("Accept")
			if strings.Contains(accept, "image/avif") && app.AutoAvif {
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
	resultKey := strings.TrimPrefix(p.Path, "meta/")
	load := func(image string) (*Blob, error) {
		return app.loadStorage(r, image)
	}
	return app.suppress(ctx, "res:"+resultKey, func(ctx context.Context) (*Blob, error) {
		if blob, err = app.loadResult(r, resultKey); err == nil && !IsBlobEmpty(blob) {
			return blob, err
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
		if IsBlobEmpty(blob) {
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
					if !IsBlobEmpty(f) {
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
		if err == nil && len(app.ResultSavers) > 0 {
			app.save(ctx, nil, app.ResultSavers, resultKey, blob)
		}
		return blob, err
	})
}

func (app *Imagor) loadStorage(r *http.Request, key string) (*Blob, error) {
	return app.suppress(r.Context(), "img:"+key, func(ctx context.Context) (blob *Blob, err error) {
		var origin Saver
		r = r.WithContext(ctx)
		blob, origin, err = app.load(r, app.Loaders, key)
		if err != nil || IsBlobEmpty(blob) {
			return
		}
		if len(app.Savers) > 0 {
			app.save(ctx, origin, app.Savers, key, blob)
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
) (blob *Blob, origin Saver, err error) {
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
		if !IsBlobEmpty(f) {
			blob = f
		}
		if e == nil {
			err = nil
			origin, _ = loader.(Saver)
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

func (app *Imagor) save(
	ctx context.Context, origin Saver, savers []Saver, key string, blob *Blob,
) {
	var cancel func()
	if app.SaveTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, app.SaveTimeout)
	}
	defer cancel()
	var wg sync.WaitGroup
	for _, saver := range savers {
		if saver == origin {
			// loaded from the same store, no need save again
			if app.Debug {
				app.Logger.Debug("skip-save", zap.String("key", key))
			}
			continue
		}
		wg.Add(1)
		go func(saver Saver) {
			defer wg.Done()
			if err := saver.Save(ctx, key, blob); err != nil {
				app.Logger.Warn("save", zap.String("key", key), zap.Error(err))
			} else if app.Debug {
				app.Logger.Debug("saved", zap.String("key", key))
			}
		}(saver)
	}
	wg.Wait()
	return
}

type suppressKey struct {
	Key string
}

func (app *Imagor) suppress(
	ctx context.Context,
	key string, fn func(ctx context.Context) (*Blob, error),
) (blob *Blob, err error) {
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
	var loaders, savers, resultLoaders, resultSavers, processors []string
	for _, v := range app.Loaders {
		loaders = append(loaders, getType(v))
	}
	for _, v := range app.Savers {
		savers = append(savers, getType(v))
	}
	for _, v := range app.Processors {
		processors = append(processors, getType(v))
	}
	for _, v := range app.ResultLoaders {
		resultLoaders = append(resultLoaders, getType(v))
	}
	for _, v := range app.ResultSavers {
		resultSavers = append(resultSavers, getType(v))
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
		zap.Strings("savers", savers),
		zap.Strings("result_loaders", resultLoaders),
		zap.Strings("result_savers", resultSavers),
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
