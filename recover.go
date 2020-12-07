package goo

import (
	"fmt"
	"net/http"
)

//Recovery recover panic
func Recovery(logger *logger) HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				if logger != nil {
					(*logger).Error(message)
				}

				c.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		c.Next()
	}
}
