package module

import (
	"flag"
	"github.com/cshum/imagor"
	"go.uber.org/zap"
)

type Callback func() (logger *zap.Logger, isDebug bool)

type Func func(fs *flag.FlagSet, cb Callback) imagor.Option

func ApplyFuncs(fs *flag.FlagSet, cb Callback, funcs ...Func) (options []imagor.Option) {
	options, _, _ = applyFuncs(fs, cb, funcs...)
	return
}

func applyFuncs(fs *flag.FlagSet, cb Callback, funcs ...Func) (options []imagor.Option, logger *zap.Logger, isDebug bool) {
	if len(funcs) == 0 {
		logger, isDebug = cb()
		return
	} else {
		var last = len(funcs) - 1
		options = append(options, funcs[last](fs, func() (*zap.Logger, bool) {
			var opts []imagor.Option
			opts, logger, isDebug = applyFuncs(fs, cb, funcs[:last]...)
			options = append(options, opts...)
			return logger, isDebug
		}))
		return
	}
}
