package floki

import (
	"github.com/julienschmidt/httprouter"
	//"html/template"
	"encoding/json"
	"log"
	"net/http"
	"os"
	//"path"
	"flag"
	"runtime"
	"sync"
)

type (
	// Used internally to collect errors that occurred during an http request.
	errorMsg struct {
		Err  string      `json:"error"`
		Type uint32      `json:"-"`
		Meta interface{} `json:"meta"`
	}

	errorMsgs []errorMsg

	Model map[string]interface{}

	// Floki represents the top level web application. inject.Injector methods can be invoked to map services on a global level.
	Floki struct {
		*RouterGroup
		logger      *log.Logger
		params      map[string]interface{}
		contextPool sync.Pool
		router      *httprouter.Router
		handlers404 []HandlerFunc
		config      map[string]*json.RawMessage
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
	f.router = httprouter.New()
	f.router.NotFound = f.handle404

	return f
}

func (f *Floki) loadConfig() {
	logger := f.logger

	var configFile string
	flag.StringVar(&configFile, "config", "../app/config/"+Env+".json", "Specify application config file to use")
	flag.Parse()

	if Env == Dev {
		logger.Println("using config file:", configFile)
	}

	err := loadConfig(configFile, &f.config)
	if err != nil {
		logger.Fatal(err)
	}

	f.logger.Println("loaded config:", f.config["AppRoot"])
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
	f.SetParameter("templates", compileTemplates(tplDir, logger))

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

func (m *Floki) GetParameter(key string) interface{} {
	return m.params[key]
}

func (m *Floki) SetParameter(key string, value interface{}) {
	m.params[key] = value
}

func (f *Floki) createContext(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Context {
	c := f.contextPool.Get().(*Context)
	c.Writer.reset(w)
	c.Request = req
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
	m := New()

	m.loadConfig()

	if Env == Dev {
		m.Use(Logger())

		if boolValue(m.config["EnableProfiling"]) {
			RegisterProfiler(m)
		}
	}

	m.Use(Recovery())

	return m
}

// Handler can be any callable function. Floki attempts to inject services into the handler's argument list.
// Floki will panic if an argument could not be fullfilled via dependency injection.
type Handler interface {
	ServeHTTP(Context)
}

func (f *Floki) Config() map[string]*json.RawMessage {
	return f.config
}

func (f *Floki) handle404(w http.ResponseWriter, req *http.Request) {
	/*handlers := f.combineHandlers(engine.handlers404)
	c := f.createContext(w, req, nil, handlers)
	c.Writer.setStatus(404)
	c.Next()
	if !c.Writer.Written() {
		c.Data(404, MIMEPlain, []byte("404 page not found"))
	}
	engine.cache.Put(c)
	*/
}
