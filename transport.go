package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	router := gin.Default()         //创建一个router
	router.Use(MiddleWare())        //中间件，起拦截器的作用
	router.Any("/*action", Forward) //所有请求都会经过Forward函数转发
	router.Run(":8000")
}

func Forward(c *gin.Context) {
	targetHost := &httputils.TargetHost{
		Host:    "www.baidu.com",
		IsHttps: false,
	}
	HostReverseProxy(c.Writer, c.Request, targetHost)
}

func HostReverseProxy(w http.ResponseWriter, req *http.Request, targetHost *TargetHost) {
	host := ""
	if targetHost.IsHttps {
		host = host + "https://"
	} else {
		host = host + "http://"
	}
	remote, err := url.Parse(host + targetHost.Host)
	if err != nil {
		log.Errorf("err:%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	if targetHost.IsHttps {
		tls, err := GetVerTLSConfig(targetHost.CAPath)
		if err != nil {
			log.Errorf("https crt error: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var pTransport http.RoundTripper = &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Second*time.Duration(DialTimeout))
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			ResponseHeaderTimeout: time.Second * time.Duration(ResponseHeaderTimeout),
			TLSClientConfig:       tls,
		}
		proxy.Transport = pTransport
	}
	proxy.ServeHTTP(w, req)
}

type TargetHost struct {
	Host    string
	IsHttps bool
	CAPath  string
}

func MiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		account := c.Request.Header.Get("ename") //从请求头中获取ename字段
		if account == "" {
			c.JSON(http.StatusOK, httputils.Response{
				Code:    400002,
				Message: "用户未登录",
			})
			c.Abort()
			return
		}
		fmt.Println("before middleware")
		c.Set("request", "clinet_request")
		c.Next()
		fmt.Println("before middleware")
	}
}
