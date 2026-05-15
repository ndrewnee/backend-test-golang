package skinport

import (
	"io"

	"github.com/andybalholm/brotli"
)

type brotliHTTPWriter struct {
	*brotli.Writer
}

func newBrotliHTTPWriter(writer io.Writer) *brotliHTTPWriter {
	return &brotliHTTPWriter{Writer: brotli.NewWriter(writer)}
}
