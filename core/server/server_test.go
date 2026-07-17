package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idena-network/idena-indexer/log"
	"github.com/stretchr/testify/require"
)

func TestHTTPServerUsesBoundedTimeouts(t *testing.T) {
	server := newHTTPServer(8080, http.NotFoundHandler())

	require.Equal(t, ":8080", server.Addr)
	require.Equal(t, readHeaderTimeout, server.ReadHeaderTimeout)
	require.Equal(t, readTimeout, server.ReadTimeout)
	require.Equal(t, writeTimeout, server.WriteTimeout)
	require.Equal(t, idleTimeout, server.IdleTimeout)
	require.Equal(t, maxHeaderBytes, server.MaxHeaderBytes)
}

func TestRequestFilterRejectsOversizedFormBody(t *testing.T) {
	server := NewServer(0, log.New())
	nextCalled := false
	handler := server.requestFilter(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))
	request := httptest.NewRequest(
		http.MethodPost,
		"http://localhost/api/test",
		strings.NewReader("value="+strings.Repeat("a", maxRequestBodyBytes)),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.False(t, nextCalled)
}

func TestRequestFilterDoesNotLogRequestMetadata(t *testing.T) {
	var output bytes.Buffer
	logger := log.New()
	logger.SetHandler(log.StreamHandler(&output, log.LogfmtFormat()))
	server := NewServer(0, logger)
	handler := server.requestFilter(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	request := httptest.NewRequest(http.MethodGet, "http://localhost/private/path?api_key=secret", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.NotContains(t, output.String(), "/private/path")
	require.NotContains(t, output.String(), "api_key")
	require.NotContains(t, output.String(), "secret")
	require.NotContains(t, output.String(), "203.0.113.10")
}
