package config

import (
	"flag"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
)

func ApplySetters(fs *flag.FlagSet, cb Callback, setters ...Setter) (options []imagor.Option) {
	options, _, _ = applySetters(fs, cb, setters...)
	return
}

func applySetters(fs *flag.FlagSet, cb Callback, setters ...Setter) (options []imagor.Option, logger *zap.Logger, isDebug bool) {
	if len(setters) == 0 {
		logger, isDebug = cb()
		return
	} else {
		var last = len(setters) - 1
		options = append(options, setters[last](fs, func() (*zap.Logger, bool) {
			opts, l, i := applySetters(fs, cb, setters[:last]...)
			options = append(options, opts...)
			logger = l
			isDebug = i
			return logger, isDebug
		}))
		return
	}
}
