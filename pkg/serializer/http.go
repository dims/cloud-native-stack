package serializer

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

// RespondJSON writes a JSON response with the given status code and data.
// It buffers the JSON encoding before writing headers to prevent partial responses.
func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	// Serialize first to detect errors before writing headers
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.Error("json encoding failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	if _, err := w.Write(buf.Bytes()); err != nil {
		// Connection is broken, log but can't recover
		slog.Warn("response write failed", "error", err)
	}
}

const (
	HttpReaderUserAgent = "CNS-Serializer/1.0"
)

var (
	HttpReaderDefaultTimeout               = 30 * time.Second
	HttpReaderDefaultKeepAlive             = 30 * time.Second
	HttpReaderDefaultConnectTimeout        = 5 * time.Second
	HttpReaderDefaultTLSHandshakeTimeout   = 5 * time.Second
	HttpReaderDefaultResponseHeaderTimeout = 10 * time.Second
	HttpReaderDefaultIdleConnTimeout       = 90 * time.Second
	HttpReaderDefaultMaxIdleConns          = 100
	HttpReaderDefaultMaxIdleConnsPerHost   = 10
	HttpReaderDefaultMaxConnsPerHost       = 0
)

// HttpReaderOption defines a configuration option for HttpReader.
type HttpReaderOption func(*HttpReader)

// HttpReader handles fetching data over HTTP with configurable options.
type HttpReader struct {
	UserAgent             string
	TotalTimeout          time.Duration
	ConnectTimeout        time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConnsPerHost       int
	InsecureSkipVerify    bool
	Client                *http.Client
}

func WithUserAgent(userAgent string) HttpReaderOption {
	return func(r *HttpReader) {
		r.UserAgent = userAgent
	}
}

func WithTotalTimeout(timeout time.Duration) HttpReaderOption {
	return func(r *HttpReader) {
		r.TotalTimeout = timeout
	}
}

func WithConnectTimeout(timeout time.Duration) HttpReaderOption {
	return func(r *HttpReader) {
		r.ConnectTimeout = timeout
	}
}

func WithTLSHandshakeTimeout(timeout time.Duration) HttpReaderOption {
	return func(r *HttpReader) {
		r.TLSHandshakeTimeout = timeout
	}
}

func WithResponseHeaderTimeout(timeout time.Duration) HttpReaderOption {
	return func(r *HttpReader) {
		r.ResponseHeaderTimeout = timeout
	}
}

func WithIdleConnTimeout(timeout time.Duration) HttpReaderOption {
	return func(r *HttpReader) {
		r.IdleConnTimeout = timeout
	}
}

func WithMaxIdleConns(max int) HttpReaderOption {
	return func(r *HttpReader) {
		r.MaxIdleConns = max
	}
}

func WithMaxIdleConnsPerHost(max int) HttpReaderOption {
	return func(r *HttpReader) {
		r.MaxIdleConnsPerHost = max
	}
}

func WithMaxConnsPerHost(max int) HttpReaderOption {
	return func(r *HttpReader) {
		r.MaxConnsPerHost = max
	}
}

func WithInsecureSkipVerify(skip bool) HttpReaderOption {
	return func(r *HttpReader) {
		r.InsecureSkipVerify = skip
	}
}

func WithClient(client *http.Client) HttpReaderOption {
	return func(r *HttpReader) {
		r.Client = client
	}
}

// NewHttpReader creates a new HttpReader with the specified options.
func NewHttpReader(options ...HttpReaderOption) *HttpReader {
	t := &http.Transport{
		// Connection pooling
		MaxIdleConns:        HttpReaderDefaultMaxIdleConns,
		MaxIdleConnsPerHost: HttpReaderDefaultMaxIdleConnsPerHost,
		MaxConnsPerHost:     HttpReaderDefaultMaxConnsPerHost,

		// Timeouts
		DialContext: (&net.Dialer{
			Timeout:   HttpReaderDefaultConnectTimeout,
			KeepAlive: HttpReaderDefaultKeepAlive,
		}).DialContext,
		TLSHandshakeTimeout:   HttpReaderDefaultTLSHandshakeTimeout,
		ResponseHeaderTimeout: HttpReaderDefaultResponseHeaderTimeout,
		ExpectContinueTimeout: 1 * time.Second,

		// Connection reuse
		IdleConnTimeout:    HttpReaderDefaultIdleConnTimeout,
		DisableKeepAlives:  false,
		DisableCompression: false,
		ForceAttemptHTTP2:  true,

		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		},
	}

	r := &HttpReader{
		UserAgent: HttpReaderUserAgent,
		Client: &http.Client{
			Timeout:   HttpReaderDefaultTimeout,
			Transport: t,
		},
	}

	// Apply options
	for _, opt := range options {
		opt(r)
	}
	return r
}

// Read fetches data from the specified URL and returns it as a byte slice.
func (r *HttpReader) Read(url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("url is empty")
	}

	// Create request
	resp, err := r.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed for url %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data: status %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Download reads data from the specified URL and writes it to the given file path.
func (r *HttpReader) Download(url, filePath string) error {
	data, err := r.Read(url)
	if err != nil {
		return fmt.Errorf("failed to read from url %s: %w", url, err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}
