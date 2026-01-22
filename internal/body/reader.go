package body

import (
	"bytes"
	"io"
)

// Reader provides body reading functionality with size limits
type Reader struct {
	maxSize int
}

// ReaderOption is an option for Reader
type ReaderOption func(*Reader)

// WithMaxSize sets the maximum body size to read
func WithMaxSize(size int) ReaderOption {
	return func(r *Reader) {
		r.maxSize = size
	}
}

// NewReader creates a new body reader
func NewReader(opts ...ReaderOption) *Reader {
	r := &Reader{
		maxSize: 64 * 1024, // 64KB default
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// ReadAndRestore reads the body up to maxSize and returns a new reader
// that can be read again. Returns the read bytes and a new io.ReadCloser.
func (r *Reader) ReadAndRestore(body io.ReadCloser) ([]byte, io.ReadCloser, error) {
	if body == nil {
		return nil, nil, nil
	}

	// Read up to maxSize + 1 to detect if body is larger
	limitedReader := io.LimitReader(body, int64(r.maxSize+1))
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, body, err
	}

	// Check if body was truncated
	truncated := len(data) > r.maxSize
	if truncated {
		data = data[:r.maxSize]
	}

	// Create a new reader that combines the read data with any remaining data
	var newBody io.ReadCloser
	if truncated {
		// Body was larger than maxSize, combine read data with remaining
		newBody = &multiReadCloser{
			Reader: io.MultiReader(bytes.NewReader(data), body),
			closer: body,
		}
	} else {
		// We read the entire body, close original and return buffered data
		body.Close()
		newBody = io.NopCloser(bytes.NewReader(data))
	}

	return data, newBody, nil
}

// ReadBytes reads the body up to maxSize and returns the bytes
// The original body will be drained and should not be used after this
func (r *Reader) ReadBytes(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	limitedReader := io.LimitReader(body, int64(r.maxSize))
	return io.ReadAll(limitedReader)
}

// multiReadCloser combines a Reader with a Closer
type multiReadCloser struct {
	io.Reader
	closer io.Closer
}

func (m *multiReadCloser) Close() error {
	return m.closer.Close()
}

// TruncatedBody wraps body bytes with truncation info
type TruncatedBody struct {
	Data      []byte
	Truncated bool
	MaxSize   int
}

// ReadWithTruncation reads the body and indicates if it was truncated
func (r *Reader) ReadWithTruncation(body io.Reader) (*TruncatedBody, error) {
	if body == nil {
		return &TruncatedBody{}, nil
	}

	// Read maxSize + 1 bytes to detect truncation
	limitedReader := io.LimitReader(body, int64(r.maxSize+1))
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	truncated := len(data) > r.maxSize
	if truncated {
		data = data[:r.maxSize]
	}

	return &TruncatedBody{
		Data:      data,
		Truncated: truncated,
		MaxSize:   r.maxSize,
	}, nil
}
