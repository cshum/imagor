package config

import (
	"flag"
	"net"

	"github.com/cshum/imagor"
	"github.com/cshum/imagor/loader/httploader"
	"go.uber.org/zap"
)

func withHTTPLoader(fs *flag.FlagSet, cb func() (*zap.Logger, bool)) imagor.Option {
	var (
		httpLoaderForwardHeaders = fs.String("http-loader-forward-headers", "",
			"Forward request header to HTTP Loader request by csv e.g. User-Agent,Accept")
		httpLoaderForwardClientHeaders = fs.Bool("http-loader-forward-client-headers", false,
			"Forward browser client request headers to HTTP Loader request")
		httpLoaderForwardAllHeaders = fs.Bool("http-loader-forward-all-headers", false,
			"Deprecated in flavour of -http-loader-forward-client-headers")
		httpLoaderAllowedSources = fs.String("http-loader-allowed-sources", "",
			"HTTP Loader allowed hosts whitelist to load images from if set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.")
		httpLoaderMaxAllowedSize = fs.Int("http-loader-max-allowed-size", 0,
			"HTTP Loader maximum allowed size in bytes for loading images if set")
		httpLoaderInsecureSkipVerifyTransport = fs.Bool("http-loader-insecure-skip-verify-transport", false,
			"HTTP Loader to use HTTP transport with InsecureSkipVerify true")
		httpLoaderDefaultScheme = fs.String("http-loader-default-scheme", "https",
			"HTTP Loader default scheme if not specified by image path. Set \"nil\" to disable default scheme.")
		httpLoaderAccept = fs.String("http-loader-accept", "*/*",
			"HTTP Loader set request Accept header and validate response Content-Type header")
		httpLoaderProxyURLs = fs.String("http-loader-proxy-urls", "",
			"HTTP Loader Proxy URLs. Enable HTTP Loader proxy only if this value present. Accept csv of proxy urls e.g. http://user:pass@host:port,http://user:pass@host:port")
		httpLoaderProxyAllowedSources = fs.String("http-loader-proxy-allowed-sources", "",
			"HTTP Loader Proxy allowed hosts that enable proxy transport, if proxy URLs are set. Accept csv wth glob pattern e.g. *.google.com,*.github.com.")
		httpLoaderBlockLoopbackNetworks = fs.Bool("http-loader-block-loopback-networks", false,
			"HTTP Loader Proxy rejects connections to loopback network IP addresses.")
		httpLoaderBlockPrivateNetworks = fs.Bool("http-loader-block-private-networks", false,
			"HTTP Loader Proxy rejects connections to private network IP addresses.")
		httpLoaderBlockLinkLocalNetworks = fs.Bool("http-loader-block-link-local-networks", false,
			"HTTP Loader Proxy rejects connections to link local network IP addresses.")
		httpLoaderBlockNetworks []*net.IPNet
		httpLoaderDisable       = fs.Bool("http-loader-disable", false,
			"Disable HTTP Loader")
	)
	fs.Var((*CIDRSliceFlag)(&httpLoaderBlockNetworks), "http-loader-block-networks",
		"HTTP Loader Proxy rejects connections to link local network IP addresses. This options takes a comma separated list of networks in CIDR notation e.g. ::1/128,127.0.0.0/8.")
	_, _ = cb()
	return func(app *imagor.Imagor) {
		if !*httpLoaderDisable {
			// fallback with HTTP Loader unless explicitly disabled
			app.Loaders = append(app.Loaders,
				httploader.New(
					httploader.WithForwardClientHeaders(
						*httpLoaderForwardClientHeaders || *httpLoaderForwardAllHeaders),
					httploader.WithAccept(*httpLoaderAccept),
					httploader.WithForwardHeaders(*httpLoaderForwardHeaders),
					httploader.WithAllowedSources(*httpLoaderAllowedSources),
					httploader.WithMaxAllowedSize(*httpLoaderMaxAllowedSize),
					httploader.WithInsecureSkipVerifyTransport(*httpLoaderInsecureSkipVerifyTransport),
					httploader.WithDefaultScheme(*httpLoaderDefaultScheme),
					httploader.WithProxyTransport(*httpLoaderProxyURLs, *httpLoaderProxyAllowedSources),
					httploader.WithBlockLoopbackNetworks(*httpLoaderBlockLoopbackNetworks),
					httploader.WithBlockPrivateNetworks(*httpLoaderBlockPrivateNetworks),
					httploader.WithBlockLinkLocalNetworks(*httpLoaderBlockLinkLocalNetworks),
					httploader.WithBlockNetworks(httpLoaderBlockNetworks...),
				),
			)
		}
	}
}
