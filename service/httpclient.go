package service

import (
	"net/http"
	"time"
)

const defaultHTTPClientTimeout = time.Minute

var defaultHTTPClient = &http.Client{
	Timeout: defaultHTTPClientTimeout,
}
