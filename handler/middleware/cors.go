package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/internal/seetie"
)

var _ NextHandler = &CORS{}

type CORS struct {
	options *CORSOptions
}

type CORSOptions struct {
	AllowedOrigins   []string
	AllowCredentials bool
	MaxAge           string
}

func NewCORSOptions(cors *config.CORS) (*CORSOptions, error) {
	if cors == nil {
		return nil, nil
	}
	dur, err := time.ParseDuration(cors.MaxAge)
	if err != nil {
		return nil, err
	}
	corsMaxAge := strconv.Itoa(int(math.Floor(dur.Seconds())))

	allowedOrigins := seetie.ValueToStringSlice(cors.AllowedOrigins)
	for i, a := range allowedOrigins {
		allowedOrigins[i] = strings.ToLower(a)
	}

	return &CORSOptions{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: cors.AllowCredentials,
		MaxAge:           corsMaxAge,
	}, nil
}

// NeedsVary if a request with not allowed origin is ignored.
func (c *CORSOptions) NeedsVary() bool {
	return !c.AllowsOrigin("*")
}

func (c *CORSOptions) AllowsOrigin(origin string) bool {
	if c == nil {
		return false
	}

	for _, a := range c.AllowedOrigins {
		if a == strings.ToLower(origin) || a == "*" {
			return true
		}
	}

	return false
}

func NewCORSHandler(opts *CORSOptions) NextHandler {
	return &CORS{
		options: opts,
	}
}

func (c *CORS) ServeNextHTTP(rw http.ResponseWriter, nextHandler http.Handler, req *http.Request) {
	if c.isCorsPreflightRequest(req) {
		c.setCorsRespHeaders(rw.Header(), req)
		rw.WriteHeader(http.StatusNoContent)
		return
	}
	c.setCorsRespHeaders(rw.Header(), req)
	nextHandler.ServeHTTP(rw, req)
}

func (c *CORS) isCorsPreflightRequest(req *http.Request) bool {
	return req.Method == http.MethodOptions &&
		(req.Header.Get("Access-Control-Request-Method") != "" ||
			req.Header.Get("Access-Control-Request-Headers") != "")
}

func (c *CORS) setCorsRespHeaders(headers http.Header, req *http.Request) {
	if !c.isCorsRequest(req) {
		return
	}
	requestOrigin := req.Header.Get("Origin")
	if !c.options.AllowsOrigin(requestOrigin) {
		return
	}
	// see https://fetch.spec.whatwg.org/#http-responses
	if c.options.AllowsOrigin("*") && !c.isCredentialed(req.Header) {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", requestOrigin)
	}

	if c.options.AllowCredentials == true {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	if c.isCorsPreflightRequest(req) {
		// Reflect request header value
		acrm := req.Header.Get("Access-Control-Request-Method")
		if acrm != "" {
			headers.Set("Access-Control-Allow-Methods", acrm)
		}
		// Reflect request header value
		acrh := req.Header.Get("Access-Control-Request-Headers")
		if acrh != "" {
			headers.Set("Access-Control-Allow-Headers", acrh)
		}
		if c.options.MaxAge != "" {
			headers.Set("Access-Control-Max-Age", c.options.MaxAge)
		}
	} else if c.options.NeedsVary() {
		headers.Add("Vary", "Origin")
	}
}

func (c *CORS) isCorsRequest(req *http.Request) bool {
	return req.Header.Get("Origin") != ""
}

func (c *CORS) isCredentialed(headers http.Header) bool {
	return headers.Get("Cookie") != "" || headers.Get("Authorization") != "" || headers.Get("Proxy-Authorization") != ""
}
