package server_test

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
)

const testdataPath = "testdata/endpoints"

func TestEndpoints_ProxyReqRes(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "01_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	logHook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.Entries
	if l := len(entries); l != 3 {
		t.Fatalf("Expected 3 log entries, given %d", l)
	}

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, given %d", res.StatusCode)
	}

	resBytes, err := ioutil.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if string(resBytes) != "808" {
		t.Errorf("Expected body 808, given %s", resBytes)
	}
}

func TestEndpoints_Res(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, logHook := newCouper(path.Join(testdataPath, "02_couper.hcl"), helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/v1", nil)
	helper.Must(err)

	logHook.Reset()

	res, err := client.Do(req)
	helper.Must(err)

	entries := logHook.Entries
	if l := len(entries); l != 1 {
		t.Fatalf("Expected 1 log entries, given %d", l)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, given %d", res.StatusCode)
	}

	resBytes, err := ioutil.ReadAll(res.Body)
	helper.Must(err)
	res.Body.Close()

	if string(resBytes) != "string" {
		t.Errorf("Expected body 'string', given %s", resBytes)
	}
}

func TestEndpoints_OAuth2(t *testing.T) {
	helper := test.New(t)
	seenCh := make(chan struct{})

	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth2" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)

			body := []byte(`{
				"access_token": "abcdef0123456789",
				"token_type": "bearer",
				"expires_in": 100
			}`)
			rw.Write(body)

			return
		}

		if req.URL.Path == "/resource" {
			if req.Header.Get("Authorization") == "Bearer abcdef0123456789" {
				rw.WriteHeader(http.StatusNoContent)

				go func() {
					seenCh <- struct{}{}
				}()

				return
			}
		}

		rw.WriteHeader(http.StatusBadRequest)
	}))
	ln, err := net.Listen("tcp4", testProxyAddr[7:])
	helper.Must(err)
	origin.Listener = ln
	origin.Start()
	defer origin.Close()
	confPath := "testdata/endpoints/03_couper.hcl"
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
	helper.Must(err)

	_, err = newClient().Do(req)
	helper.Must(err)

	timer := time.NewTimer(time.Second)
	select {
	case <-timer.C:
		t.Error("OAuth2 request failed")
	case <-seenCh:
	}
}
