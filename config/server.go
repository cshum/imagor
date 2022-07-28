package config

import (
	"flag"
	"fmt"
	"github.com/cshum/imagor"
	"github.com/cshum/imagor/server"
	"github.com/peterbourgon/ff/v3"
	"go.uber.org/zap"
	"runtime"
)

func CreateServer(args []string, funcs ...Func) (srv *server.Server) {
	var (
		fs     = flag.NewFlagSet("imagor", flag.ExitOnError)
		logger *zap.Logger
		err    error
		app    *imagor.Imagor

		debug        = fs.Bool("debug", false, "Debug mode")
		version      = fs.Bool("version", false, "Imagor version")
		port         = fs.Int("port", 8000, "Sever port")
		goMaxProcess = fs.Int("gomaxprocs", 0, "GOMAXPROCS")

		_ = fs.String("config", ".env", "Retrieve configuration from the given file")

		serverAddress = fs.String("server-address", "",
			"Server address")
		serverPathPrefix = fs.String("server-path-prefix", "",
			"Server path prefix")
		serverCORS = fs.Bool("server-cors", false,
			"Enable CORS")
		serverStripQueryString = fs.Bool("server-strip-query-string", false,
			"Enable strip query string redirection")
		serverAccessLog = fs.Bool("server-access-log", false,
			"Enable server access log")
	)

	app = NewImagor(fs, func() (*zap.Logger, bool) {
		if err = ff.Parse(fs, args,
			ff.WithEnvVars(),
			ff.WithConfigFileFlag("config"),
			ff.WithIgnoreUndefined(true),
			ff.WithAllowMissingConfigFile(true),
			ff.WithConfigFileParser(ff.EnvParser),
		); err != nil {
			panic(err)
		}
		if *debug {
			if logger, err = zap.NewDevelopment(); err != nil {
				panic(err)
			}
		} else {
			if logger, err = zap.NewProduction(); err != nil {
				panic(err)
			}
		}
		return logger, *debug
	}, append(funcs, WithFile, WithHTTPLoader)...)

	if *version {
		fmt.Println(imagor.Version)
		return
	}

	if *goMaxProcess > 0 {
		logger.Debug("GOMAXPROCS", zap.Int("count", *goMaxProcess))
		runtime.GOMAXPROCS(*goMaxProcess)
	}

	return server.New(app,
		server.WithAddress(*serverAddress),
		server.WithPort(*port),
		server.WithPathPrefix(*serverPathPrefix),
		server.WithCORS(*serverCORS),
		server.WithStripQueryString(*serverStripQueryString),
		server.WithAccessLog(*serverAccessLog),
		server.WithLogger(logger),
		server.WithDebug(*debug),
	)
}
