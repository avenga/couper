package handler

import (
	"context"
	"net/http"
	"net/http/httputil"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
)

// Proxy wraps a httputil.ReverseProxy to apply additional configuration context
// and have control over the roundtrip configuration.
type Proxy struct {
	backend      http.RoundTripper
	context      hcl.Body
	evalCtx      *eval.HTTP
	reverseProxy *httputil.ReverseProxy
}

func NewProxy(backend http.RoundTripper, ctx hcl.Body, evalCtx *eval.HTTP) *Proxy {
	proxy := &Proxy{
		backend: backend,
		context: ctx,
		evalCtx: evalCtx,
	}
	rp := &httputil.ReverseProxy{
		Director:  proxy.director,
		Transport: backend,
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, err error) {
			if rec, ok := rw.(*transport.Recorder); ok {
				rec.SetError(err)
			}
		},
	}
	proxy.reverseProxy = rp
	return proxy
}

func (p *Proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := eval.ApplyRequestContext(p.evalCtx, p.context, req); err != nil {
		return nil, err // TODO: log only
	}

	*req = *req.WithContext(context.WithValue(req.Context(), request.RoundTripProxy, true))

	rec := transport.NewRecorder()
	p.reverseProxy.ServeHTTP(rec, req)
	beresp, err := rec.Response(req)
	if err != nil {
		return beresp, err
	}
	err = eval.ApplyResponseContext(p.evalCtx, p.context, req, beresp) // TODO: log only
	return beresp, err
}

func (p *Proxy) director(_ *http.Request) {
	// noop
}
