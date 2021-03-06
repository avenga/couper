package logging

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

type RoundtripHandlerFunc http.HandlerFunc

func (f RoundtripHandlerFunc) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	f(rw, req)
}

type AccessLog struct {
	conf   *Config
	logger logrus.FieldLogger
}

func NewAccessLog(c *Config, logger logrus.FieldLogger) *AccessLog {
	return &AccessLog{
		conf:   c,
		logger: logger,
	}
}

func (log *AccessLog) ServeHTTP(rw http.ResponseWriter, req *http.Request, nextHandler http.Handler, startTime time.Time) {
	statusRecorder := NewStatusRecorder(rw)
	rw = statusRecorder

	nextHandler.ServeHTTP(rw, req)
	serveDone := time.Now()

	fields := Fields{
		"proto": req.Proto,
	}

	backendName, _ := req.Context().Value(request.BackendName).(string)
	if backendName == "" {
		endpointName, _ := req.Context().Value(request.Endpoint).(string)
		fields["endpoint"] = endpointName
	}

	fields["method"] = req.Method
	fields["server"] = req.Context().Value(request.ServerName)
	fields["uid"] = req.Context().Value(request.UID)

	requestFields := Fields{
		"headers": filterHeader(log.conf.RequestHeaders, req.Header),
	}
	fields["request"] = requestFields

	if req.ContentLength > 0 {
		requestFields["bytes"] = req.ContentLength
	}

	// Read out handler kind from stringer interface
	if h, ok := nextHandler.(fmt.Stringer); ok && h.String() != "" {
		fields["handler"] = h.String()
	} else if kind, ok := req.Context().Value(request.EndpointKind).(string); ok { // fallback, e.g. with ErrorHandler
		fields["handler"] = kind
	}

	if log.conf.TypeFieldKey != "" {
		fields["type"] = log.conf.TypeFieldKey
	}

	path := &url.URL{
		Path:       req.URL.Path,
		RawPath:    req.URL.RawPath,
		RawQuery:   req.URL.RawQuery,
		ForceQuery: req.URL.ForceQuery,
		Fragment:   req.URL.Fragment,
	}
	requestFields["path"] = path.String()

	if req.Host != "" {
		requestFields["addr"] = req.Host
		requestFields["host"], requestFields["port"] = splitHostPort(req.Host)
	}

	if req.URL.User != nil && req.URL.User.Username() != "" {
		fields["auth_user"] = req.URL.User.Username()
	} else if user, _, ok := req.BasicAuth(); ok && user != "" {
		fields["auth_user"] = user
	}

	fields["realtime"] = roundMS(serveDone.Sub(startTime))
	fields["status"] = statusRecorder.status

	responseFields := Fields{
		"headers": filterHeader(log.conf.ResponseHeaders, rw.Header()),
	}
	fields["response"] = responseFields

	if statusRecorder.writtenBytes > 0 {
		responseFields["bytes"] = statusRecorder.writtenBytes
	}

	requestFields["tls"] = req.TLS != nil
	fields["scheme"] = "http"
	if req.URL.Scheme != "" {
		fields["scheme"] = req.URL.Scheme
	}
	if requestFields["port"] == "" {
		if fields["scheme"] == "https" {
			requestFields["port"] = "443"
		} else {
			requestFields["port"] = "80"
		}
	}

	fields["url"] = fields["scheme"].(string) + "://" + req.Host + path.String()

	var err error
	fields["client_ip"], _ = splitHostPort(req.RemoteAddr)
	if couperErr := statusRecorder.Header().Get(errors.HeaderErrorCode); couperErr != "" {
		i, _ := strconv.Atoi(couperErr[:4])
		err = errors.Code(i)
		fields["code"] = i
	}

	var entry *logrus.Entry
	if log.conf.ParentFieldKey != "" {
		entry = log.logger.WithField(log.conf.ParentFieldKey, fields)
	} else {
		entry = log.logger.WithFields(logrus.Fields(fields))
	}
	entry.Time = startTime

	if statusRecorder.status == http.StatusInternalServerError || err != nil {
		if err != nil {
			entry.Error(err)
			return
		}
		entry.Error()
	} else {
		entry.Info()
	}
}
