package goo

import (
	"context"
	"html/template"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

type logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Fatal(v ...interface{})
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

//RouterGroup router group
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	parent      *RouterGroup
	engine      *Engine
}

// HandlerFunc defines the request handler used by goo
type HandlerFunc func(*Context)

// Engine implement the interface of ServeHTTP
type Engine struct {
	*RouterGroup
	router        *router
	groups        []*RouterGroup
	htmlTemplates *template.Template //html render
	funcMap       template.FuncMap   //html render
	logger        logger
	server        *http.Server

	ctx context.Context //ctx 上下文
	wg  *sync.WaitGroup //wg 计数君
}

// New is the constructor of goo.Engine
func New() *Engine {
	engine := &Engine{router: newRouter(), logger: nil}
	engine.server = &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       300 * time.Second,
		MaxHeaderBytes:    1024 << 10,
	}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	engine.Use(Recovery(&engine.logger))
	return engine
}

//SetLogger 设置日志logger
func (engine *Engine) SetLogger(logger logger) *Engine {
	engine.logger = logger
	return engine
}

//SetServer 设置server属性
func (engine *Engine) SetServer(server *http.Server) *Engine {
	engine.server = server
	return engine
}

//SetContext 优雅关闭服务;
//nums: 关闭服务器超时时间/s;
func (engine *Engine) SetContext(ctx context.Context, wg *sync.WaitGroup, nums int) *Engine {

	engine.ctx = ctx
	engine.wg = wg

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), time.Duration(nums)*time.Second)
		defer cancel()

		engine.logger.Info("等待服务器关闭...")
		if err := engine.server.Shutdown(c); err != nil {
			engine.logger.Error(err)
		}
	}()
	return engine
}

//SetFuncMap SetFuncMap
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

//LoadHTMLGlob LoadHTMLGlob
func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filePath")

		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

//Static Static file
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

//Group Group of router
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	group.engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// PUT defines the method to add PUT request
func (group *RouterGroup) PUT(pattern string, handler HandlerFunc) {
	group.addRoute("PUT", pattern, handler)
}

// DELETE defines the method to add DELETE request
func (group *RouterGroup) DELETE(pattern string, handler HandlerFunc) {
	group.addRoute("DELETE", pattern, handler)
}

// Request  Request
func (group *RouterGroup) Request(Request, pattern string, handler HandlerFunc) {
	group.addRoute(Request, pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) error {
	engine.server.Addr = addr
	engine.server.Handler = engine

	engine.logger.Info("服务器已启动", addr)
	return engine.server.ListenAndServe()
}

//Use add middlewares
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	engine.logger.Info(req.Method, req.URL.Path)
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}

	c := newContext(w, req)
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c)
}
