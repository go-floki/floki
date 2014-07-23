package floki

import (
	"net/http/pprof"
)

func RegisterProfiler(m *Floki) {

	m.logger.Println("profiling enabled. url: /debug/pprof/")

	m.GET("/debug/pprof/:id", func(c *Context) {
		c.Logger().Println("id: ", c.Param("id"))
		//		pprof.Index(c.Writer, c.Request)
		switch c.Param("id") {
		case "cmdline":
			pprof.Cmdline(c.Writer, c.Request)
			break
		case "profile":
			pprof.Profile(c.Writer, c.Request)
			break
		case "symbol":
			pprof.Symbol(c.Writer, c.Request)
			break
		default:
			pprof.Index(c.Writer, c.Request)
		}

	})

	m.GET("/debug/pprof/", func(c *Context) {
		pprof.Index(c.Writer, c.Request)
	})

}
