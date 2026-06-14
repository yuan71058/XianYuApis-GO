package httpclient

import "net/http"

// Middleware HTTP 中间件函数类型。
type Middleware func(req *http.Request) error

// ChainMiddlewares 将多个中间件串联。
func ChainMiddlewares(middlewares ...Middleware) Middleware {
	return func(req *http.Request) error {
		for _, mw := range middlewares {
			if err := mw(req); err != nil {
				return err
			}
		}
		return nil
	}
}
