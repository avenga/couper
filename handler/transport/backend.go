package transport

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	couperErr "github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/logging"
)

const (
	GzipName              = "gzip"
	AcceptEncodingHeader  = "Accept-Encoding"
	ContentEncodingHeader = "Content-Encoding"
	ContentLengthHeader   = "Content-Length"
	VaryHeader            = "Vary"
)

var _ http.RoundTripper = &Backend{}

var ReClientSupportsGZ = regexp.MustCompile(`(?i)\b` + GzipName + `\b`)

// Backend represents the transport configuration.
type Backend struct {
	context          hcl.Body
	name             string
	openAPIValidator *validation.OpenAPI
	options          *BackendOptions
	transportConf    *Config
	upstreamLog      *logging.UpstreamLog
	// oauth
	// ...
	// TODO: OrderedList for origin AC, middlewares etc.
}

// NewBackend creates a new <*Backend> object by the given <*Config>.
func NewBackend(ctx hcl.Body, tc *Config, opts *BackendOptions, log *logrus.Entry) http.RoundTripper {
	logEntry := log
	if tc.BackendName != "" {
		logEntry = log.WithField("backend", tc.BackendName)
	}

	var openAPI *validation.OpenAPI
	if opts != nil {
		openAPI = validation.NewOpenAPI(opts.OpenAPI)
	}

	backend := &Backend{
		context:          ctx,
		openAPIValidator: openAPI,
		options:          opts,
		transportConf:    tc,
	}
	backend.upstreamLog = logging.NewUpstreamLog(logEntry, backend, tc.NoProxyFromEnv)
	return backend.upstreamLog
}

// RoundTrip implements the <http.RoundTripper> interface.
func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
	tc := b.evalTransport(req)
	t := Get(tc)

	if b.transportConf.Timeout > 0 {
		deadline, cancel := context.WithTimeout(req.Context(), b.transportConf.Timeout)
		defer cancel()
		*req = *req.WithContext(deadline)
	}

	if req.URL.Scheme == "" {
		req.URL.Scheme = tc.Scheme
	}

	req.URL.Host = tc.Origin
	req.Host = tc.Hostname

	err := eval.ApplyRequestContext(req.Context(), b.context, req)
	if err != nil {
		return nil, err
	}

	// oauth ....

	if b.options != nil && b.options.BasicAuth != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(b.options.BasicAuth))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	// handler.Proxy marks proxy roundtrips since we should not handle headers twice.
	_, isProxyReq := req.Context().Value(request.RoundTripProxy).(bool)

	if !isProxyReq {
		removeConnectionHeaders(req.Header)
		removeHopHeaders(req.Header)
	}

	if ReClientSupportsGZ.MatchString(req.Header.Get(AcceptEncodingHeader)) {
		req.Header.Set(AcceptEncodingHeader, GzipName)
	} else {
		req.Header.Del(AcceptEncodingHeader)
	}

	if b.openAPIValidator != nil {
		if err = b.openAPIValidator.ValidateRequest(req); err != nil {
			return nil, couperErr.UpstreamRequestValidationFailed
		}
	}

	setUserAgent(req)
	req.Close = false
	beresp, err := t.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if b.openAPIValidator != nil {
		if err = b.openAPIValidator.ValidateResponse(beresp); err != nil {
			return nil, couperErr.UpstreamResponseValidationFailed
		}
	}

	if strings.ToLower(beresp.Header.Get(ContentEncodingHeader)) == GzipName {
		src, rerr := gzip.NewReader(beresp.Body)
		if rerr == nil {
			beresp.Header.Del(ContentEncodingHeader)
			beresp.Body = eval.NewReadCloser(src, beresp.Body)
		}
	}

	if !isProxyReq {
		removeConnectionHeaders(req.Header)
	}

	// Backend response context creates the beresp variables in first place and applies this context
	// to the current beresp obj. Downstream response context evals reading their beresp variable values
	// from this result. Fallback case is for testing purposes.
	if evalCtx, ok := req.Context().Value(eval.ContextType).(*eval.Context); ok {
		evalCtx = evalCtx.WithBeresps(beresp)
		err = eval.ApplyResponseContext(evalCtx, b.context, beresp)
	} else {
		err = eval.ApplyResponseContext(req.Context(), b.context, beresp)
	}

	return beresp, err
}

func (b *Backend) evalTransport(req *http.Request) *Config {
	var httpContext *hcl.EvalContext
	if httpCtx, ok := req.Context().Value(eval.ContextType).(*eval.Context); ok {
		httpContext = httpCtx.HCLContext()
	}

	content, _, diags := b.context.PartialContent(config.BackendInlineSchema)
	if diags.HasErrors() {
		b.upstreamLog.LogEntry().Error(diags)
	}

	var origin, hostname string
	if o := getAttribute(httpContext, "origin", content); o != "" {
		origin = o
	}
	if h := getAttribute(httpContext, "hostname", content); h != "" {
		hostname = h
	}

	originURL, _ := url.Parse(origin)
	if hostname == "" {
		hostname = originURL.Host
	}

	return b.transportConf.With(originURL.Scheme, originURL.Host, hostname)
}

func getAttribute(ctx *hcl.EvalContext, name string, body *hcl.BodyContent) string {
	attr := body.Attributes
	if _, ok := attr[name]; !ok {
		return ""
	}
	originValue, _ := attr[name].Expr.Value(ctx)
	return seetie.ValueToString(originValue)
}

// setUserAgent sets an empty one if none is present or empty
// to prevent the go http defaultUA gets written.
func setUserAgent(outreq *http.Request) {
	if ua := outreq.Header.Get("User-Agent"); ua == "" {
		outreq.Header.Set("User-Agent", "")
	}
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func removeHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		hv := header.Get(h)
		if hv == "" {
			continue
		}
		if h == "Te" && hv == "trailers" {
			// Issue 21096: tell backend applications that
			// care about trailer support that we support
			// trailers. (We do, but we don't go out of
			// our way to advertise that unless the
			// incoming client request thought it was
			// worth mentioning)
			continue
		}
		header.Del(h)
	}
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}
