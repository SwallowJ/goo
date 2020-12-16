package goo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

//H is H
type H map[string]interface{}

//Context 上下文
type Context struct {
	//origin info
	Writer http.ResponseWriter
	Req    *http.Request

	//request info
	Path   string
	Method string
	Params map[string]string

	//response info
	StatusCode int

	handlers []HandlerFunc
	index    int
	engine   *Engine
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Path:   req.URL.Path,
		Method: req.Method,
		Req:    req,
		Writer: w,
		index:  -1,
	}
}

//AddWait 添加groutine 计数
func (c *Context) AddWait(num int) {
	c.engine.wg.Add(num)
}

//Down goroutine 执行结束
func (c *Context) Down() {
	c.engine.wg.Done()
}

//GetContext 获取上下文
func (c *Context) GetContext() context.Context {
	return c.engine.ctx
}

//Next middleWare
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

//Fail Fail
func (c *Context) Fail(code int, err string) {
	c.index = len(c.handlers)
	c.JSON(code, H{"message": err})
	if c.engine.logger != nil {
		c.engine.logger.Error(code, "--", c.Req.Method, c.Req.URL.Path)
	}
}

//Param get Params
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

//PostForm get FormValue
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

//Query get Query params
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

//Status write StatusCode
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
	if c.engine.logger != nil {
		if code == http.StatusOK {
			c.engine.logger.Info(code, "--", c.Req.Method, c.Req.URL.Path)
		} else {
			c.engine.logger.Error(code, "--", c.Req.Method, c.Req.URL.Path)
		}
	}
}

//SetHeader set response Header
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

//String write string response
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

//JSON write json response
func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

//Data write data response
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

//HTML response html template
func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(http.StatusInternalServerError, err.Error())
	}
}
