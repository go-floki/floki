package floki

import (
	//	"log"
	//"fmt"
	"net/http"
	"time"
)

// Logger returns a middleware handler that logs the request as it goes in and the response as it goes out.
func Logger() HandlerFunc {
	return HandlerFunc(func(c *Context) {
		res := c.Writer
		req := c.Request

		start := time.Now()

		addr := req.Header.Get("X-Real-IP")
		if addr == "" {
			addr = req.Header.Get("X-Forwarded-For")
			if addr == "" {
				addr = req.RemoteAddr
			}
		}

		log := c.Logger()

		log.Printf("Started %s %s for %s", req.Method, req.URL.Path, addr)

		rw := res.(ResponseWriter)
		c.Next()

		log.Printf("Completed %v %s in %v\n", rw.Status(), http.StatusText(rw.Status()), time.Since(start))
	})
}
