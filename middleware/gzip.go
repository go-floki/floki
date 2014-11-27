package middleware

import (
	"compress/gzip"
	"github.com/go-floki/floki"
	"mime"
	"path/filepath"
	"strings"
)

const (
	encodingGzip = "gzip"

	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentLength   = "Content-Length"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"

	BestCompression    = gzip.BestCompression
	BestSpeed          = gzip.BestSpeed
	DefaultCompression = gzip.DefaultCompression
	NoCompression      = gzip.NoCompression
)

type gzipWriter struct {
	floki.ResponseWriter
	gzwriter *gzip.Writer
}

func newGzipWriter(writer floki.ResponseWriter, gzwriter *gzip.Writer) *gzipWriter {
	return &gzipWriter{writer, gzwriter}
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.gzwriter.Write(data)
}

func GzipMiddleware(level int) floki.HandlerFunc {
	return func(c *floki.Context) {
		req := c.Request
		if !strings.Contains(req.Header.Get(headerAcceptEncoding), encodingGzip) {
			c.Next()
			return
		}

		// don't compress some mime types
		mimeType := mime.TypeByExtension(filepath.Ext(c.Request.RequestURI))
		if strings.Contains(mimeType, "image") {
			c.Next()
			return
		}

		writer := c.Writer
		gz, err := gzip.NewWriterLevel(writer, level)
		if err != nil {
			c.Next()
			return
		}

		headers := writer.Header()
		headers.Set(headerContentEncoding, encodingGzip)
		headers.Set(headerVary, headerAcceptEncoding)
		//		headers.Set("Content-type", "text/html; charset=utf-8")

		gzwriter := newGzipWriter(c.Writer, gz)

		oldWriter := c.Writer
		c.Writer = gzwriter

		putOldWriterBack := func() {
			gz.Close()
			c.Writer = oldWriter
		}

		defer putOldWriterBack()

		c.Next()

		writer.Header().Del(headerContentLength)
	}
}
