package transport_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
)

func TestBackend_RoundTrip_Timings(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead {
			time.Sleep(time.Second * 2) // > ttfb proxy settings
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []struct {
		name           string
		context        hcl.Body
		tconf          *transport.Config
		req            *http.Request
		expectedStatus int
	}{
		{"with zero timings", test.NewRemainContext("origin", origin.URL), &transport.Config{}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), http.StatusNoContent},
		{"with overall timeout", test.NewRemainContext("origin", "http://1.2.3.4/"), &transport.Config{Timeout: time.Second}, httptest.NewRequest(http.MethodGet, "http://1.2.3.5/", nil), http.StatusBadGateway},
		{"with connect timeout", test.NewRemainContext("origin", "http://blackhole.webpagetest.org/"), &transport.Config{ConnectTimeout: time.Second}, httptest.NewRequest(http.MethodGet, "http://1.2.3.6/", nil), http.StatusBadGateway},
		{"with ttfb timeout", test.NewRemainContext("origin", origin.URL), &transport.Config{TTFBTimeout: time.Second}, httptest.NewRequest(http.MethodHead, "http://1.2.3.7/", nil), http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			log := logger.WithContext(context.Background())

			backend := transport.NewBackend(eval.NewENVContext(nil), tt.context, tt.tconf, log, nil)
			proxy := handler.NewProxy(backend, tt.context, eval.NewENVContext(nil))

			hook.Reset()

			res, err := proxy.RoundTrip(tt.req)
			if err != nil {
				t.Error(err)
				return
			}

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectedStatus, res.StatusCode)
			} else {
				return // no error log for expected codes
			}
		})
	}
}

func TestBackend_Compression_Disabled(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Accept-Encoding") != "" {
			t.Error("Unexpected Accept-Encoding header")
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	u := seetie.GoToValue(origin.URL)
	hclBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	})
	backend := transport.NewBackend(eval.NewENVContext(nil), hclBody, &transport.Config{
		Origin: origin.URL,
	}, log, nil)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	res, err := backend.RoundTrip(req)
	helper.Must(err)

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("Expected 204, got: %d", res.StatusCode)
	}
}

func TestBackend_Compression_ModifyAcceptEncoding(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if ae := req.Header.Get("Accept-Encoding"); ae != "gzip" {
			t.Errorf("Unexpected Accept-Encoding header: %s", ae)
		}

		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		for i := 1; i < 1000; i++ {
			w.Write([]byte("<html/>"))
		}
		w.Close()

		rw.Header().Set("Content-Encoding", "gzip")
		rw.Write(b.Bytes())
	}))
	defer origin.Close()

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	u := seetie.GoToValue(origin.URL)
	hclBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	})

	backend := transport.NewBackend(eval.NewENVContext(nil), hclBody, &transport.Config{
		Origin: origin.URL,
	}, log, nil)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	res, err := backend.RoundTrip(req)
	helper.Must(err)

	if l := res.Header.Get("Content-Length"); l != "60" {
		t.Errorf("Unexpected C/L: %s", l)
	}

	n, err := io.Copy(ioutil.Discard, res.Body)
	helper.Must(err)

	if n != 6993 {
		t.Errorf("Unexpected body length: %d, want: %d", n, 6993)
	}
}
