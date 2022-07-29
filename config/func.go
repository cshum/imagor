package config

import (
	"flag"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
)

type Func func(fs *flag.FlagSet, cb func() (logger *zap.Logger, isDebug bool)) imagor.Option

func applyFuncs(
	fs *flag.FlagSet, cb func() (*zap.Logger, bool), funcs ...Func,
) (options []imagor.Option, logger *zap.Logger, isDebug bool) {
	if len(funcs) == 0 {
		logger, isDebug = cb()
		return
	} else {
		var last = len(funcs) - 1
		var called bool
		if funcs[last] == nil {
			return applyFuncs(fs, cb, funcs[:last]...)
		}
		options = append(options, funcs[last](fs, func() (*zap.Logger, bool) {
			options, logger, isDebug = applyFuncs(fs, cb, funcs[:last]...)
			called = true
			return logger, isDebug
		}))
		if !called {
			var opts []imagor.Option
			opts, logger, isDebug = applyFuncs(fs, cb, funcs[:last]...)
			options = append(opts, options...)
			return options, logger, isDebug
		}
		return
	}
}
