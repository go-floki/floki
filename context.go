package floki

import (
	"encoding/json"
	"errors"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"log"
	"math"
	"net/http"
)

const (
	AbortIndex = math.MaxInt8 / 2
)

const (
	ErrorTypeInternal = 1 << iota
	ErrorTypeExternal = 1 << iota
	ErrorTypeAll      = 0xffffffff
)

const (
	MIMEJSON  = "application/json"
	MIMEHTML  = "text/html"
	MIMEXML   = "application/xml"
	MIMEXML2  = "text/xml"
	MIMEPlain = "text/plain"
)

type (
	BeforeFunc func(*Context)

	// Context is the most important part of floki. It allows us to pass variables between middleware,
	// manage the flow, validate the JSON of a request and render a JSON response for example.
	Context struct {
		Request     *http.Request
		Writer      ResponseWriter
		Keys        map[string]interface{}
		Errors      errorMsgs
		Params      httprouter.Params
		Floki       *Floki
		handlers    []HandlerFunc
		index       int8
		beforeFuncs []BeforeFunc
	}
)

func (c *Context) Copy() *Context {
	var cp Context = *c
	cp.index = AbortIndex
	cp.handlers = nil
	cp.beforeFuncs = nil
	return &cp
}

// Next should be used only in the middlewares.
// It executes the pending handlers in the chain inside the calling handler.
// See example in github.
func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Forces the system to do not continue calling the pending handlers.
// For example, the first handler checks if the request is authorized. If it's not, context.Abort(401) should be called.
// The rest of pending handlers would never be called for that request.
func (c *Context) Abort(code int) {
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.index = AbortIndex
}

func (c *Context) Logger() *log.Logger {
	return c.Floki.logger
}

func (c *Context) Param(name string) interface{} {
	return c.Params.ByName(name)
}

func (c *Context) Render(tplName string, data Model) {
	c.Writer.Header().Set("Content-Type", MIMEHTML)

	templates := c.Floki.GetParameter("templates").(map[string]*template.Template)
	tpl := templates[tplName]

	if tpl != nil {
		// populate model with context variables
		for key, value := range c.Keys {
			data[key] = value
		}

		tpl.Execute(c.Writer, data)
	}
}

func (c *Context) Json(code int, data interface{}) {
	writer := c.Writer
	c.Logger().Println("headers:", c.Writer.Header())
	writer.Header().Set("Content-type", "application/json")
	c.Logger().Println("headers:", c.Writer.Header())
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	encoder.Encode(data)

	//_, err := writer.Write([]byte())
}

func (c *Context) Send(code int, response string) error {
	writer := c.Writer
	writer.WriteHeader(code)
	_, err := writer.Write([]byte(response))
	return err
}

func (c *Context) Redirect(urlStr string) {
	c.RedirectWith(302, urlStr)
}

func (c *Context) RedirectWith(code int, urlStr string) {
	c.Writer.Header().Set("Location", urlStr)
	c.Writer.WriteHeader(code)
}

// Fail is the same as Abort plus an error message.
// Calling `context.Fail(500, err)` is equivalent to:
// ```
// context.Error("Operation aborted", err)
// context.Abort(500)
// ```
func (c *Context) Fail(code int, err error) {
	c.Error(err, "Operation aborted")
	c.Abort(code)
}

func (c *Context) ErrorTyped(err error, typ uint32, meta interface{}) {
	c.Errors = append(c.Errors, errorMsg{
		Err:  err.Error(),
		Type: typ,
		Meta: meta,
	})
}

// Attaches an error to the current context. The error is pushed to a list of errors.
// It's a good idea to call Error for each error that occurred during the resolution of a request.
// A middleware can be used to collect all the errors and push them to a database together, print a log, or append it in the HTTP response.
func (c *Context) Error(err error, meta interface{}) {
	c.ErrorTyped(err, ErrorTypeExternal, meta)
}

/************************************/
/******** METADATA MANAGEMENT********/
/************************************/

// Sets a new pair key/value just for the specified context.
// It also lazy initializes the hashmap.
func (c *Context) Set(key string, item interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = item
}

// Get returns the value for the given key or an error if the key does not exist.
func (c *Context) Get(key string) (interface{}, error) {
	if c.Keys != nil {
		item, ok := c.Keys[key]
		if ok {
			return item, nil
		}
	}
	return nil, errors.New("Key does not exist.")
}

// MustGet returns the value for the given key or panics if the value doesn't exist.
func (c *Context) MustGet(key string) interface{} {
	value, err := c.Get(key)
	if err != nil || value == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return value
}

func (c *Context) BeforeDestroy(before BeforeFunc) {
	c.beforeFuncs = append(c.beforeFuncs, before)
}

// This is executed before context object is freed
func (c *Context) beforeRelease() {
	for i := len(c.beforeFuncs) - 1; i >= 0; i-- {
		c.beforeFuncs[i](c)
	}
}
