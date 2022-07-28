package config

import (
	"flag"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
)

func ApplyFlagFuncs(fs *flag.FlagSet, cb Callback, funcs ...FlagFunc) (options []imagor.Option) {
	options, _, _ = applyFlagFuncs(fs, cb, funcs...)
	return
}

func applyFlagFuncs(fs *flag.FlagSet, cb Callback, funcs ...FlagFunc) (options []imagor.Option, logger *zap.Logger, isDebug bool) {
	if len(funcs) == 0 {
		logger, isDebug = cb()
		return
	} else {
		var last = len(funcs) - 1
		options = append(options, funcs[last](fs, func() (*zap.Logger, bool) {
			var opts []imagor.Option
			opts, logger, isDebug = applyFlagFuncs(fs, cb, funcs[:last]...)
			options = append(options, opts...)
			return logger, isDebug
		}))
		return
	}
}
