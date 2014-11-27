package floki

import (
	"github.com/go-floki/router"
	//"html/template"
	"log"
	"net/http"
	"os"
	//"path"
	"flag"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var appEventHandlers map[string][]AppEventHandler = make(map[string][]AppEventHandler)
var TimeZone *time.Location

type (
	// Used internally to collect errors that occurred during an http request.
	errorMsg struct {
		Err  string      `json:"error"`
		Type uint32      `json:"-"`
		Meta interface{} `json:"meta"`
	}

	errorMsgs []errorMsg

	Model map[string]interface{}

	AppEventHandler func(app *Floki)

	// Floki represents the top level web application. inject.Injector methods can be invoked to map services on a global level.
	Floki struct {
		*RouterGroup
		logger      *log.Logger
		params      map[string]interface{}
		contextPool sync.Pool
		router      *router.Router
		handlers404 []HandlerFunc

		Config   ConfigMap
		TimeZone *time.Location
	}

	// Used internally to configure router, a RouterGroup is associated with a prefix
	// and an array of handlers (middlewares)
	RouterGroup struct {
		Handlers []HandlerFunc
		prefix   string
		parent   *RouterGroup
		floki    *Floki
	}

	HandlerFunc func(*Context)
)

// New creates a bare bones Floki instance. Use this method if you want to have full control over the middleware that is used.
func New() *Floki {
	f := &Floki{}
	f.RouterGroup = &RouterGroup{nil, "/", nil, f}
	f.logger = log.New(os.Stdout, "[floki] ", 0)
	f.params = make(map[string]interface{})
	f.contextPool.New = func() interface{} {
		return &Context{Floki: f, Writer: &responseWriter{}}
	}
	f.router = router.New()
	f.router.NotFound = f.handle404

	return f
}

func (f *Floki) loadConfig() {
	logger := f.logger

	f.triggerAppEvent("ConfigureAppStart")

	var configFile string
	flag.StringVar(&configFile, "config", "../app/config/"+Env+".json", "Specify application config file to use")
	flag.Parse()

	if Env == Dev {
		logger.Println("using config file:", configFile)
	}

	err := loadConfig(configFile, &f.Config)
	if err != nil {
		logger.Fatal(err)
	}

	timeZoneStr := f.Config.Str("timeZone", "")
	TimeZone, err = time.LoadLocation(timeZoneStr)
	if err != nil {
		logger.Println("Invalid timezone in configuration file specified:", timeZoneStr, ". Falling back to UTC")
		TimeZone, err = time.LoadLocation("")
	}

	f.triggerAppEvent("ConfigureAppEnd")

	f.logger.Println("loaded config:", configFile)
}

// ServeHTTP is the HTTP Entry point for a Floki instance. Useful if you want to control your own HTTP server.
func (f *Floki) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	f.router.ServeHTTP(res, req)
}

// Run the http server. Listening on os.GetEnv("PORT") or 3000 by default.
func (f *Floki) Run() {
	logger := f.logger

	if Env == Prod {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	tplDir := f.GetParameter("views dir").(string)
	f.SetParameter("templates", f.compileTemplates(tplDir, logger))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	host := os.Getenv("HOST")
	_ = host

	addr := host + ":" + port

	logger.Printf("listening on %s (%s)\n", addr, Env)

	if err := http.ListenAndServe(addr, f); err != nil {
		panic(err)
	}
}

func (f *Floki) GetParameter(key string) interface{} {
	return f.params[key]
}

func (f *Floki) SetParameter(key string, value interface{}) {
	f.params[key] = value
}

func (f *Floki) createContext(w http.ResponseWriter, req *http.Request, params router.Params, handlers []HandlerFunc) *Context {
	c := f.contextPool.Get().(*Context)
	c.Writer.reset(w)
	c.Request = req
	f.initContext(c, params, handlers)
	return c
}

func (f *Floki) initContext(c *Context, params router.Params, handlers []HandlerFunc) *Context {
	c.Params = params
	c.handlers = handlers
	c.Keys = nil
	c.index = -1
	c.beforeFuncs = nil
	return c
}

// Classic creates a classic Floki with some basic default middleware - floki.Logger, floki.Recovery and floki.Static.
// Classic also maps floki.Routes as a service.
func Default() *Floki {
	f := New()

	f.loadConfig()

	if Env == Dev {
		f.Use(Logger())
	}

	if f.Config.Bool("enableProfiling", false) {
		RegisterProfiler(f)
	}

	f.Use(Recovery())

	return f
}

// Handler can be any callable function. Floki attempts to inject services into the handler's argument list.
// Floki will panic if an argument could not be fullfilled via dependency injection.
type Handler interface {
	ServeHTTP(Context)
}

func (f *Floki) handle404(w http.ResponseWriter, req *http.Request) {
	//handlers := f.combineHandlers(f.handlers404)
	handlers := f.handlers404

	c := f.createContext(w, req, nil, handlers)
	writer := c.Writer
	writer.setStatus(404)
	c.Next()

	if !writer.Written() {
		writer.Header().Set("Content-type", MIMEHTML)
		_404response := "<html><body><h1>Error 404</h1><p>Page not found</p></body></html>"
		writer.Header().Set("Content-length", strconv.Itoa(len(_404response)))
		c.Send(404, _404response)
	}

	f.contextPool.Put(c)

}

func RegisterAppEventHandler(event string, handler AppEventHandler) {
	handlers, exists := appEventHandlers[event]

	if exists {
		handlers = append(handlers, handler)
	} else {
		handlers = make([]AppEventHandler, 8)
		handlers = append(handlers[0:0], handler)
	}

	appEventHandlers[event] = handlers

}

func (f *Floki) Logger() *log.Logger {
	return f.logger
}

func (f *Floki) triggerAppEvent(event string) {
	handlers, exists := appEventHandlers[event]
	if exists {
		for idx := range handlers {
			//f.logger.Println("trigger handler", idx, "of event", event, handlers)
			handlers[idx](f)
		}
	}
}

// Adds not found handlers
func (f *Floki) Handle404(handlers ...HandlerFunc) {
	f.handlers404 = append(f.handlers404, handlers...)
}
