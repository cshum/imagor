package imagor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cshum/imagor/imagorpath"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
)

// Version imagor version
const Version = "1.6.15"

// Loader image loader interface
type Loader interface {
	Get(r *http.Request, key string) (*Blob, error)
}

// Storage image storage interface
type Storage interface {
	// Get data Blob by key
	Get(r *http.Request, key string) (*Blob, error)

	// Stat get Blob Stat by key
	Stat(ctx context.Context, key string) (*Stat, error)

	// Put data Blob by key
	Put(ctx context.Context, key string, blob *Blob) error

	// Delete delete data Blob by key
	Delete(ctx context.Context, key string) error
}

// Stater optional interface for loaders that support stat operations
type Stater interface {
	Stat(ctx context.Context, key string) (*Stat, error)
}

// LoadFunc function handler for Processor to call loader
type LoadFunc func(string) (*Blob, error)

// Processor process image buffer
type Processor interface {
	// Startup processor startup lifecycle,
	// called only once for the application lifetime
	Startup(ctx context.Context) error

	// Process Blob with given params and loader function
	Process(ctx context.Context, blob *Blob, params imagorpath.Params, load LoadFunc) (*Blob, error)

	// Shutdown processor shutdown lifecycle,
	// called only once for the application lifetime
	Shutdown(ctx context.Context) error
}

// Imagor main application
type Imagor struct {
	Unsafe                 bool
	Signer                 imagorpath.Signer
	StoragePathStyle       imagorpath.StorageHasher
	ResultStoragePathStyle imagorpath.ResultStorageHasher
	BasePathRedirect       string
	Loaders                []Loader
	Storages               []Storage
	ResultStorages         []Storage
	Processors             []Processor
	RequestTimeout         time.Duration
	LoadTimeout            time.Duration
	SaveTimeout            time.Duration
	ProcessTimeout         time.Duration
	CacheHeaderTTL         time.Duration
	CacheHeaderSWR         time.Duration
	ProcessConcurrency     int64
	ProcessQueueSize       int64
	AutoWebP               bool
	AutoAVIF               bool
	AutoJPEG               bool
	ModifiedTimeCheck      bool
	DisableErrorBody       bool
	DisableParamsEndpoint  bool
	EnablePostRequests     bool
	ResponseRawOnError     bool
	BaseParams             string
	Logger                 *zap.Logger
	Debug                  bool

	g          singleflight.Group
	sema       *semaphore.Weighted
	queueSema  *semaphore.Weighted
	baseParams imagorpath.Params
}

// New create new Imagor
func New(options ...Option) *Imagor {
	app := &Imagor{
		Logger:         zap.NewNop(),
		RequestTimeout: time.Second * 30,
		LoadTimeout:    time.Second * 20,
		SaveTimeout:    time.Second * 20,
		ProcessTimeout: time.Second * 20,
		CacheHeaderTTL: time.Hour * 24 * 7,
		CacheHeaderSWR: time.Hour * 24,
	}
	for _, option := range options {
		option(app)
	}
	if app.ProcessConcurrency > 0 {
		app.sema = semaphore.NewWeighted(app.ProcessConcurrency)
		app.queueSema = semaphore.NewWeighted(app.ProcessQueueSize + app.ProcessConcurrency)
	}
	if app.Debug {
		app.debugLog()
	}
	if app.Signer == nil {
		app.Signer = imagorpath.NewDefaultSigner("")
	}
	app.BaseParams = strings.TrimSpace(app.BaseParams)
	if app.BaseParams != "" {
		app.BaseParams = strings.TrimSuffix(app.BaseParams, "/") + "/"
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

// ServeHTTP implements http.Handler for imagor operations
func (app *Imagor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Handle POST requests only when unsafe mode and POST requests are enabled
	if r.Method == http.MethodPost {
		if !app.Unsafe || !app.EnablePostRequests {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		app.handlePostRequest(w, r)
		return
	}
	path := r.URL.EscapedPath()
	if path == "/" || path == "" {
		if app.BasePathRedirect == "" {
			renderLandingPage(w)
		} else {
			http.Redirect(w, r, app.BasePathRedirect, http.StatusTemporaryRedirect)
		}
		return
	}

	// Check if this is a GET request to a processing path with no image
	p := imagorpath.Parse(path)
	if p.Image == "" && !p.Params && app.EnablePostRequests && app.Unsafe {
		// Show upload form for processing paths when POST requests are enabled
		renderUploadForm(w, path)
		return
	}
	if p.Params {
		if !app.DisableParamsEndpoint {
			writeJSONIndent(w, r, p)
		}
		return
	}
	blob, err := checkBlob(app.Do(r, p))
	if err == ErrInvalid || err == ErrSignatureMismatch {
		if path2, e := url.QueryUnescape(path); e == nil {
			path = path2
			p = imagorpath.Parse(path)
			blob, err = checkBlob(app.Do(r, p))
		}
	}
	if err != nil {
		// Check if we should respond with raw image on error
		if app.ResponseRawOnError && !isBlobEmpty(blob) {
			e := WrapError(err)
			app.Logger.Warn("response-raw-on-error",
				zap.Any("params", p),
				zap.Error(err),
				zap.Int("status", e.Code))

			// Write error status code but serve raw image
			w.WriteHeader(e.Code)
			app.setResponseHeaders(w, r, blob, p)
			reader, size, _ := blob.NewReader()
			writeBody(w, r, reader, size)
			return
		}

		app.handleErrorResponse(w, r, err)
		return
	}
	if isBlobEmpty(blob) {
		return
	}
	app.setResponseHeaders(w, r, blob, p)
	if blob != nil && checkStatNotModified(w, r, blob.Stat) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	reader, size, _ := blob.NewReader()
	writeBody(w, r, reader, size)
	return
}

// Serve serves imagor by context and params
func (app *Imagor) Serve(ctx context.Context, p imagorpath.Params) (*Blob, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	if err != nil {
		return nil, err
	}
	p.Path = "" // make sure path generated
	return app.Do(r, p)
}

// ServeBlob serves imagor Blob with context and params, skipping loader and storages
func (app *Imagor) ServeBlob(
	ctx context.Context, blob *Blob, p imagorpath.Params,
) (*Blob, error) {
	if ctx == nil || blob == nil {
		return nil, errors.New("imagor: nil context blob")
	}
	ctx = withContext(ctx)
	mustContextRef(ctx).Blob = blob
	p.Image = "" // make sure blob is used
	return app.Serve(ctx, p)
}

// Do executes imagor operations
func (app *Imagor) Do(r *http.Request, p imagorpath.Params) (blob *Blob, err error) {
	var ctx = withContext(r.Context())
	var cancel func()
	if app.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, app.RequestTimeout)
		contextDefer(ctx, cancel)
		r = r.WithContext(ctx)
	}
	if !(app.Unsafe && p.Unsafe) && app.Signer != nil && p.Path != "" {
		if hash := app.Signer.Sign(p.Path); hash != p.Hash {
			err = ErrSignatureMismatch
			if app.Debug {
				app.Logger.Debug("sign-mismatch", zap.Any("params", p), zap.String("expected", hash))
			}
			return
		}
	}
	var isPathChanged bool
	if app.BaseParams != "" {
		p = imagorpath.Apply(p, app.BaseParams)
		isPathChanged = true
	}
	var hasFormat, hasPreview, isRaw bool
	var filters = p.Filters
	p.Filters = nil

	for _, f := range filters {
		switch f.Name {
		case "expire":
			// expire(timestamp) filter
			if ts, e := strconv.ParseInt(f.Args, 10, 64); e == nil {
				if exp := time.UnixMilli(ts); !exp.IsZero() && time.Now().After(exp) {
					err = ErrExpired
					return
				}
				r.Header.Set("Cache-Control", "private")
			}
		case "format":
			hasFormat = true
		case "raw":
			r.Header.Set("Imagor-Raw", "1")
			isRaw = true
		case "preview":
			r.Header.Set("Cache-Control", "no-cache")
			hasPreview = true // disable result storage on preview() filter
		}
		// exclude utility filters from result path
		switch f.Name {
		case "expire", "attachment":
			isPathChanged = true
		default:
			p.Filters = append(p.Filters, f)
		}
	}
	// auto WebP / AVIF / JPEG
	if !hasFormat && (app.AutoWebP || app.AutoAVIF || app.AutoJPEG) {
		accept := r.Header.Get("Accept")
		if app.AutoAVIF && strings.Contains(accept, "image/avif") {
			p.Filters = append(p.Filters, imagorpath.Filter{
				Name: "format",
				Args: "avif",
			})
			r.Header.Set("Imagor-Auto-Format", "avif") // response Vary: Accept header
			isPathChanged = true
		} else if app.AutoWebP && strings.Contains(accept, "image/webp") {
			p.Filters = append(p.Filters, imagorpath.Filter{
				Name: "format",
				Args: "webp",
			})
			r.Header.Set("Imagor-Auto-Format", "webp") // response Vary: Accept header
			isPathChanged = true
		} else if app.AutoJPEG && (accept == "" || strings.Contains(accept, "image/jpeg") || strings.Contains(accept, "image/*") || strings.Contains(accept, "*/*")) {
			p.Filters = append(p.Filters, imagorpath.Filter{
				Name: "format",
				Args: "jpeg",
			})
			r.Header.Set("Imagor-Auto-Format", "jpeg") // response Vary: Accept header
			isPathChanged = true
		}
	}
	if isPathChanged || p.Path == "" {
		p.Path = imagorpath.GeneratePath(p)
	}
	if p.Width < 0 {
		p.Width = -p.Width
		p.HFlip = !p.HFlip
	}
	if p.Height < 0 {
		p.Height = -p.Height
		p.VFlip = !p.VFlip
	}
	var resultKey string
	if p.Image != "" && !hasPreview {
		if app.ResultStoragePathStyle != nil {
			resultKey = app.ResultStoragePathStyle.HashResult(p)
		} else {
			resultKey = p.Path
		}
	}
	load := func(image string) (*Blob, error) {
		blob, _, err := app.loadStorage(r, image)
		return blob, err
	}
	return app.suppress(ctx, resultKey, func(ctx context.Context, cb func(*Blob, error)) (*Blob, error) {
		if resultKey != "" && !isRaw {
			if blob := app.loadResult(r, resultKey, p.Image); blob != nil {
				return blob, nil
			}
		}
		if app.queueSema != nil && !isRaw {
			if !app.queueSema.TryAcquire(1) {
				err = ErrTooManyRequests
				if app.Debug {
					app.Logger.Debug("queue-acquire", zap.Error(err))
				}
				return blob, err
			}
			defer app.queueSema.Release(1)
		}
		if app.sema != nil && !isRaw {
			if err = app.sema.Acquire(ctx, 1); err != nil {
				if app.Debug {
					app.Logger.Debug("acquire", zap.Error(err))
				}
				return blob, err
			}
			defer app.sema.Release(1)
		}
		var shouldSave bool
		if blob, shouldSave, err = app.loadStorage(r, p.Image); err != nil {
			if app.Debug {
				app.Logger.Debug("load", zap.Any("params", p), zap.Error(err))
			}
			return blob, err
		}

		sourceBlob := blob
		var doneSave chan struct{}
		if shouldSave {
			doneSave = make(chan struct{})
			var storageKey = p.Image
			if app.StoragePathStyle != nil {
				storageKey = app.StoragePathStyle.Hash(p.Image)
			}
			go func(blob *Blob) {
				app.saveWithErrorHandling(ctx, app.Storages, storageKey, blob)
				close(doneSave)
			}(blob)
		}
		if isBlobEmpty(blob) {
			return blob, err
		}
		if !isRaw {
			var cancel func()
			if app.ProcessTimeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, app.ProcessTimeout)
				contextDefer(ctx, cancel)
			}
			var forwardP = p
			for _, processor := range app.Processors {
				b, e := checkBlob(processor.Process(ctx, blob, forwardP, load))
				if !isBlobEmpty(b) {
					if blob != nil && blob.Header != nil && b.Header == nil {
						b.Header = blob.Header // forward blob Header
					}
					blob = b // forward Blob to next processor if exists
				}
				if e == nil {
					blob = b
					err = nil
					if app.Debug {
						app.Logger.Debug("processed", zap.Any("params", forwardP))
					}
					break
				} else if forward, ok := e.(ErrForward); ok {
					err = e
					forwardP = forward.Params
					if app.Debug {
						app.Logger.Debug("forward", zap.Any("params", forwardP))
					}
				} else {
					if ctx.Err() == nil {
						err = e
						app.Logger.Warn("process", zap.Any("params", p), zap.Error(err))
					} else {
						err = ctx.Err()
					}
					break
				}
			}
		}
		if shouldSave {
			// make sure storage saved before response and result storage
			<-doneSave
		}
		cb(blob, err)
		ctx = detachContext(ctx)
		if err == nil && !isBlobEmpty(blob) && resultKey != "" && !isRaw &&
			len(app.ResultStorages) > 0 {
			app.saveWithErrorHandling(ctx, app.ResultStorages, resultKey, blob)
		}
		if err != nil && shouldSave {
			var storageKey = p.Image
			if app.StoragePathStyle != nil {
				storageKey = app.StoragePathStyle.Hash(p.Image)
			}
			app.del(ctx, app.Storages, storageKey)
		}

		// Release fanout resources early when safe to do so
		// Only release when processing created a new blob (source != result)
		// Release the SOURCE blob instead of final processed blob
		if err == nil && !isBlobEmpty(sourceBlob) &&
			blob != sourceBlob && // Only release if processing created new blob
			!shouldSave && // Source won't be saved to storage
			!isRaw {
			_ = sourceBlob.Release()
		}

		return blob, err
	})
}

// handlePostRequest handles POST upload requests
func (app *Imagor) handlePostRequest(w http.ResponseWriter, r *http.Request) {
	// Use imagorpath to parse URL path for processing parameters
	path := r.URL.EscapedPath()
	if path == "/" || path == "" {
		path = "/" // Default path for uploads without processing
	}

	// Parse imagor parameters from URL path
	p := imagorpath.Parse(path)

	// Set image to empty string to indicate upload source (no source key)
	p.Image = ""
	p.Unsafe = true // POST uploads are always unsafe

	// Process the upload through normal imagor pipeline
	blob, err := checkBlob(app.Do(r, p))
	if err != nil {
		app.handleErrorResponse(w, r, err)
		return
	}

	if isBlobEmpty(blob) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Set response headers
	app.setResponseHeaders(w, r, blob, p)

	reader, size, _ := blob.NewReader()
	writeBody(w, r, reader, size)
}

func (app *Imagor) requestWithLoadContext(r *http.Request) *http.Request {
	var ctx = r.Context()
	var cancel func()
	if app.LoadTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, app.LoadTimeout)
		contextDefer(ctx, cancel)
		return r.WithContext(ctx)
	}
	return r
}

func (app *Imagor) loadResult(r *http.Request, resultKey, imageKey string) *Blob {
	r = app.requestWithLoadContext(r)
	ctx := r.Context()
	blob, origin, err := fromStorages(r, app.ResultStorages, resultKey)
	if err == nil && !isBlobEmpty(blob) {
		if app.ModifiedTimeCheck && origin != nil && blob.Stat != nil {
			var sourceStat *Stat
			var sourceStatErr error

			// Try loader stat first (if no Storages configured)
			if len(app.Storages) == 0 {
				sourceStat, sourceStatErr = app.loaderStat(ctx, imageKey)
				if app.Debug {
					if sourceStatErr == nil && sourceStat != nil {
						app.Logger.Debug("source stat from loader succeeded",
							zap.String("image_key", imageKey),
							zap.Time("source_time", sourceStat.ModifiedTime))
					} else {
						app.Logger.Debug("source stat from loader failed",
							zap.String("image_key", imageKey),
							zap.Error(sourceStatErr))
					}
				}
			}

			// Fallback to storage stat if loader didn't work or Storages is configured
			if (sourceStat == nil || sourceStatErr != nil) && len(app.Storages) > 0 {
				sourceStat, sourceStatErr = app.storageStat(ctx, imageKey)
			}

			if sourceStat != nil && sourceStatErr == nil {
				if app.Debug {
					app.Logger.Debug("modified-time-check",
						zap.Time("result_time", blob.Stat.ModifiedTime),
						zap.Time("source_time", sourceStat.ModifiedTime),
						zap.Bool("result_before_source", blob.Stat.ModifiedTime.Before(sourceStat.ModifiedTime)),
						zap.String("result_key", resultKey),
						zap.String("image_key", imageKey))
				}
				if !blob.Stat.ModifiedTime.Before(sourceStat.ModifiedTime) {
					return blob
				}
			} else {
				if app.Debug {
					app.Logger.Debug("modified-time-check-failed-fallback-to-cache",
						zap.Bool("has_source_stat", sourceStat != nil),
						zap.Error(sourceStatErr),
						zap.String("image_key", imageKey))
				}
				// If we can't stat the source, use the cached result
				// This handles cases where source is in loader but not storage
				return blob
			}
		} else {
			if app.Debug && app.ModifiedTimeCheck {
				app.Logger.Debug("modified-time-check-skipped",
					zap.Bool("has_origin", origin != nil),
					zap.Bool("has_blob_stat", blob.Stat != nil),
					zap.String("result_key", resultKey))
			}
			return blob
		}
	}
	return nil
}

func fromStorages(
	r *http.Request, storages []Storage, key string,
) (blob *Blob, origin Storage, err error) {
	for _, storage := range storages {
		b, e := checkBlob(storage.Get(r, key))
		if !isBlobEmpty(b) {
			blob = b
			if e == nil {
				err = nil
				origin = storage
				return
			}
		}
		err = e
	}
	return
}

func (app *Imagor) loadStorage(r *http.Request, key string) (blob *Blob, shouldSave bool, err error) {
	r = app.requestWithLoadContext(r)
	var origin Storage
	blob, origin, err = app.fromStoragesAndLoaders(r, app.Storages, app.Loaders, key)
	if !isBlobEmpty(blob) && origin == nil &&
		key != "" && err == nil && len(app.Storages) > 0 {
		shouldSave = true
	}
	return
}

func (app *Imagor) fromStoragesAndLoaders(
	r *http.Request, storages []Storage, loaders []Loader, image string,
) (blob *Blob, origin Storage, err error) {
	if image == "" {
		ref := mustContextRef(r.Context())
		if ref.Blob == nil {
			// For POST uploads, try loaders even with empty image key
			if r.Method == http.MethodPost {
				for _, loader := range loaders {
					b, e := checkBlob(loader.Get(r, image))
					if !isBlobEmpty(b) {
						blob = b
						if e == nil {
							err = nil
							return
						}
					}
					err = e
				}
			}
			if err == nil && isBlobEmpty(blob) {
				err = ErrNotFound
			}
		} else {
			blob = ref.Blob
		}
		return
	}
	var storageKey = image
	if app.StoragePathStyle != nil {
		storageKey = app.StoragePathStyle.Hash(image)
	}
	if storageKey != "" {
		blob, origin, err = fromStorages(r, storages, storageKey)
		if !isBlobEmpty(blob) && origin != nil && err == nil {
			return
		}
	}
	for _, loader := range loaders {
		b, e := checkBlob(loader.Get(r, image))
		if !isBlobEmpty(b) {
			blob = b
			if e == nil {
				err = nil
				return
			}
		}
		err = e
	}
	if err == nil && isBlobEmpty(blob) {
		err = ErrNotFound
	}
	return
}

func (app *Imagor) storageStat(ctx context.Context, key string) (stat *Stat, err error) {
	for _, storage := range app.Storages {
		if stat, err = storage.Stat(ctx, key); stat != nil && err == nil {
			return
		}
	}
	return
}

func (app *Imagor) loaderStat(ctx context.Context, key string) (stat *Stat, err error) {
	for _, loader := range app.Loaders {
		if stater, ok := loader.(Stater); ok {
			if stat, err = stater.Stat(ctx, key); stat != nil && err == nil {
				return
			}
		}
	}
	return
}

// saveWithErrorHandling saves blob to storage with cleanup on error
func (app *Imagor) saveWithErrorHandling(ctx context.Context, storages []Storage, key string, blob *Blob) {
	if key == "" {
		return
	}
	if app.SaveTimeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, app.SaveTimeout)
		defer cancel()
	}
	var wg sync.WaitGroup
	for _, storage := range storages {
		wg.Add(1)
		go func(storage Storage) {
			defer wg.Done()
			if err := storage.Put(ctx, key, blob); err != nil {
				app.Logger.Warn("save", zap.String("key", key), zap.Error(err))
				if delErr := storage.Delete(ctx, key); delErr != nil {
					app.Logger.Warn("delete-after-save-error",
						zap.String("key", key), zap.Error(delErr))
				} else if app.Debug {
					app.Logger.Debug("deleted-after-save-error", zap.String("key", key))
				}
			} else if app.Debug {
				app.Logger.Debug("saved", zap.String("key", key))
			}
		}(storage)
	}
	wg.Wait()
}

func (app *Imagor) del(ctx context.Context, storages []Storage, key string) {
	ctx = detachContext(ctx)
	if app.SaveTimeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, app.SaveTimeout)
		defer cancel()
	}
	var wg sync.WaitGroup
	for _, storage := range storages {
		wg.Add(1)
		go func(storage Storage) {
			defer wg.Done()
			if err := storage.Delete(ctx, key); err != nil {
				app.Logger.Warn("delete", zap.String("key", key), zap.Error(err))
			} else if app.Debug {
				app.Logger.Debug("deleted", zap.String("key", key))
			}
		}(storage)
	}
	wg.Wait()
	return
}

type suppressKey struct {
	Key string
}

func blobNoop(*Blob, error) {}

func (app *Imagor) suppress(
	ctx context.Context,
	key string, fn func(ctx context.Context, cb func(*Blob, error)) (*Blob, error),
) (blob *Blob, err error) {
	if key == "" {
		return fn(ctx, blobNoop)
	}
	if app.Debug {
		app.Logger.Debug("suppress", zap.String("key", key))
	}
	if isAcquired, ok := ctx.Value(suppressKey{key}).(bool); ok && isAcquired {
		// resolve deadlock
		return fn(ctx, blobNoop)
	}
	chanCb := make(chan singleflight.Result, 1)
	cb := func(blob *Blob, err error) {
		chanCb <- singleflight.Result{Val: blob, Err: err}
	}
	isCanceled := false
	ch := app.g.DoChan(key, func() (v interface{}, err error) {
		v, err = fn(context.WithValue(ctx, suppressKey{key}, true), cb)
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
	case res := <-chanCb:
		return res.Val.(*Blob), res.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// setResponseHeaders sets common response headers for blob responses
func (app *Imagor) setResponseHeaders(w http.ResponseWriter, r *http.Request, blob *Blob, p imagorpath.Params) {
	if blob == nil {
		w.Header().Set("Content-Type", "application/octet-stream")
		return
	}
	w.Header().Set("Content-Type", blob.ContentType())
	w.Header().Set("Content-Disposition", getContentDisposition(p, blob))
	setCacheHeaders(w, r, getTtl(p, app.CacheHeaderTTL), app.CacheHeaderSWR)

	if r.Header.Get("Imagor-Auto-Format") != "" {
		w.Header().Add("Vary", "Accept")
	}
	if r.Header.Get("Imagor-Raw") != "" {
		w.Header().Set("Content-Security-Policy", "script-src 'none'")
	}
	if h := blob.Header; h != nil {
		for key := range h {
			w.Header().Set(key, h.Get(key))
		}
	}
}

// handleErrorResponse handles error responses consistently across endpoints
func (app *Imagor) handleErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.Canceled) {
		w.WriteHeader(499)
		return
	}
	e := WrapError(err)
	if app.DisableErrorBody {
		w.WriteHeader(e.Code)
		return
	}
	w.WriteHeader(e.Code)
	writeJSON(w, r, e)
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

func checkStatNotModified(w http.ResponseWriter, r *http.Request, stat *Stat) bool {
	if stat == nil || strings.Contains(r.Header.Get("Cache-Control"), "no-cache") {
		return false
	}
	var isETagMatch, isNotModified bool
	var etag = stat.ETag
	if etag == "" && stat.Size > 0 && !stat.ModifiedTime.IsZero() {
		etag = fmt.Sprintf(
			"%x-%x", int(stat.ModifiedTime.Unix()), int(stat.Size))
	}
	if etag != "" {
		w.Header().Set("ETag", etag)
		if inm := r.Header.Get("If-None-Match"); inm == etag {
			isETagMatch = true
		}
	}
	if mTime := stat.ModifiedTime; !mTime.IsZero() {
		w.Header().Set("Last-Modified", mTime.Format(http.TimeFormat))
		if ims := r.Header.Get("If-Modified-Since"); ims != "" {
			if imsTime, err := time.Parse(http.TimeFormat, ims); err == nil {
				isNotModified = mTime.Before(imsTime)
			}
		}
		if !isNotModified {
			if ius := r.Header.Get("If-Unmodified-Since"); ius != "" {
				if iusTime, err := time.Parse(http.TimeFormat, ius); err == nil {
					isNotModified = mTime.After(iusTime)
				}
			}
		}
	}
	return isETagMatch || isNotModified
}

func getTtl(p imagorpath.Params, defaultTtl time.Duration) time.Duration {
	for _, f := range p.Filters {
		if f.Name == "expire" {
			if ts, e := strconv.ParseInt(f.Args, 10, 64); e == nil {
				ttl := (time.UnixMilli(ts).Sub(time.Now()) + time.Second - 1).Truncate(time.Second)
				if ttl <= defaultTtl {
					return ttl
				}
			}
		}
	}
	return defaultTtl
}

func setCacheHeaders(w http.ResponseWriter, r *http.Request, ttl, swr time.Duration) {
	if strings.Contains(r.Header.Get("Cache-Control"), "no-cache") {
		ttl = 0
	}
	expires := time.Now().Add(ttl)
	isPrivate := strings.Contains(r.Header.Get("Cache-Control"), "private")
	w.Header().Add("Expires", strings.Replace(expires.Format(time.RFC1123), "UTC", "GMT", -1))
	w.Header().Add("Cache-Control", getCacheControl(isPrivate, ttl, swr))
}

func getCacheControl(isPrivate bool, ttl, swr time.Duration) string {
	if ttl == 0 {
		return "private, no-cache, no-store, must-revalidate"
	}
	var ttlSec = int64(ttl.Seconds())
	var val = fmt.Sprintf("public, s-maxage=%d", ttlSec)
	if isPrivate {
		val = "private"
	}
	val += fmt.Sprintf(", max-age=%d, no-transform", ttlSec)
	if swr > 0 && swr < ttl {
		val += fmt.Sprintf(", stale-while-revalidate=%d", int64(swr.Seconds()))
	}
	return val
}

func writeJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf)
	}
	return
}

func writeJSONIndent(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf, _ := json.MarshalIndent(v, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf)
	}
	return
}

func writeBody(w http.ResponseWriter, r *http.Request, reader io.ReadCloser, size int64) {
	defer func() {
		_ = reader.Close()
	}()
	if size > 0 {
		// total size known, use io.Copy
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		if r.Method != http.MethodHead {
			_, _ = io.Copy(w, reader)
		}
	} else {
		// total size unknown, read all
		buf, _ := io.ReadAll(reader)
		w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
		if r.Method != http.MethodHead {
			_, _ = w.Write(buf)
		}
	}
}

func getContentDisposition(p imagorpath.Params, blob *Blob) string {
	for _, f := range p.Filters {
		if f.Name == "attachment" {
			filename := f.Args
			if filename == "" {
				_, filename = filepath.Split(p.Image)
			}
			filename = strings.ReplaceAll(filename, `"`, "%22")
			if blob != nil {
				if ext := getExtension(blob.BlobType()); ext != "" &&
					!(ext == ".jpg" && strings.HasSuffix(filename, ".jpeg")) {
					filename = strings.TrimSuffix(filename, ext) + ext
				}
			}
			return fmt.Sprintf(`attachment; filename="%s"`, filename)
		}
	}
	return "inline"
}

func getType(v interface{}) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}
