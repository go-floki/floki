package floki

import (
	"github.com/go-floki/router"
	"net/http"
	"path"
	"time"
)

type RouteHandler struct {
	floki            *Floki
	path             string
	handlers         []HandlerFunc
	handlersCombined []HandlerFunc
}

func (r RouteHandler) Handle(w http.ResponseWriter, req *http.Request, params router.Params) {
	c := r.floki.createContext(w, req, params, r.handlersCombined)
	c.Next()
	c.beforeRelease()
	r.floki.contextPool.Put(c)
}

func (r RouteHandler) HandleWithContext(c *Context, params router.Params) {
	c2 := r.floki.createContext(c.Writer, c.Request, params, r.handlers)
	c2.Next()
	c2.beforeRelease()
	r.floki.contextPool.Put(c2)
}

// Handle registers a new request handle and middlewares with the given path and method.
// The last handler should be the real handler, the other ones should be middlewares that can and should be shared among different routes.
// See the example code in github.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (group *RouterGroup) Handle(method, p string, handlers []HandlerFunc) {
	p = path.Join(group.prefix, p)

	rh := RouteHandler{
		group.floki,
		p,
		handlers,
		group.combineHandlers(handlers),
	}

	group.floki.router.Handle(method, p, rh)
}

// POST is a shortcut for router.Handle("POST", path, handle)
func (group *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	group.Handle("POST", path, handlers)
}

// GET is a shortcut for router.Handle("GET", path, handle)
func (group *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	group.Handle("GET", path, handlers)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (group *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	group.Handle("DELETE", path, handlers)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle)
func (group *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	group.Handle("PATCH", path, handlers)
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func (group *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	group.Handle("PUT", path, handlers)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle)
func (group *RouterGroup) OPTIONS(path string, handlers ...HandlerFunc) {
	group.Handle("OPTIONS", path, handlers)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle)
func (group *RouterGroup) HEAD(path string, handlers ...HandlerFunc) {
	group.Handle("HEAD", path, handlers)
}

// Static serves files from the given file system root.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use :
//     router.Static("/static", "/var/www")
func (group *RouterGroup) Static(p, root string) {
	p = path.Join(p, "/*filepath")
	fileServer := http.FileServer(http.Dir(root))

	group.GET(p, func(c *Context) {
		original := c.Request.URL.Path
		c.Request.URL.Path = c.Params.ByName("filepath")

		writer := c.Writer

		headers := writer.Header()

		// in production environment static content needs to be cached by browsers & proxies
		if Env == Prod {
			// cache for 3 months
			headers.Set("Expires", time.Now().AddDate(0, 3, 0).Format(http.TimeFormat))

			headers.Add("Cache-Control", "public")
			headers.Add("Cache-Control", "max-age=28771200")
		}

		fileServer.ServeHTTP(writer, c.Request)
		c.Request.URL.Path = original
	})
}

func (group *RouterGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(group.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, group.Handlers...)
	h = append(h, handlers...)
	return h
}

// Adds middlewares to the group, see example code in github.
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.Handlers = append(group.Handlers, middlewares...)
}

// Creates a new router group. You should add all the routes that have common middlwares or the same path prefix.
// For example, all the routes that use a common middlware for authorization could be grouped.
func (group *RouterGroup) Group(component string, handlers ...HandlerFunc) *RouterGroup {
	prefix := path.Join(group.prefix, component)
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		parent:   group,
		prefix:   prefix,
		floki:    group.floki,
	}
}
