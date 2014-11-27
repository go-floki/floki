package middleware

import (
	"compress/gzip"
	"github.com/go-floki/floki"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

const (
	testResponse = "Gzip Test Response "
)

func newServer(useGzip bool) *floki.Floki {
	r := floki.New()

	if useGzip {
		r.Use(GzipMiddleware(DefaultCompression))
	}

	r.GET("/", func(c *floki.Context) {
		c.Writer.Header().Set(headerContentLength, strconv.Itoa(len(testResponse)))
		c.Send(200, testResponse)
	})

	return r
}

func TestGzip(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(headerAcceptEncoding, encodingGzip)

	w := httptest.NewRecorder()

	r := newServer(true)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d. Location: %s", http.StatusOK, w.Code, w.HeaderMap.Get("Location"))
	}

	enc := w.Header().Get(headerContentEncoding)
	if enc != encodingGzip {
		t.Errorf("Error Header %s", enc)
	}

	cType := w.Header().Get(headerContentType)
	if cType != floki.MIMEPlain {
		t.Errorf("Error content-type: %s", cType)
	}

	enc = w.Header().Get(headerVary)
	if enc != headerAcceptEncoding {
		t.Errorf("Error Header %s", enc)
	}

	length := w.Header().Get(headerContentLength)
	if length != "" {
		t.Errorf("Error Header %s", length)
	}

	if w.Body.Len() == 19 {
		t.Fail()
	}

	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	body, _ := ioutil.ReadAll(gr)
	if string(body) != testResponse {
		t.Fail()
	}

}

func TestNoGzip(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add(headerAcceptEncoding, encodingGzip)

	w := httptest.NewRecorder()

	r := newServer(false)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code should be %v, was %d. Location: %s", http.StatusOK, w.Code, w.HeaderMap.Get("Location"))
	}

	enc := w.Header().Get(headerContentEncoding)
	if enc != "" {
		t.Errorf("Error Header %s", enc)
	}

	length := w.Header().Get(headerContentLength)
	if length != "19" {
		t.Errorf("Error Header %s", length)
	}
	if w.Body.String() != testResponse {
		t.Fail()
	}

}
