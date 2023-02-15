package transmit

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"simple_proxygateway/logger"
	"strings"
	"time"

	"simple_proxygateway/etcd"
)

type transmitHandler interface {
	getUrlString(req *http.Request) string
}

var transmitHandlerMap = make(map[string]transmitHandler)

func NewProxyHandler(serviceDiscover etcd.ServiceDiscover) http.Handler {

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			u, _ := url.Parse("https://www.qq.com/")
			req.URL = u
			req.Host = u.Host // 必须显示修改Host，否则转发可能失败
			fmt.Println(u.Host)
		},
		ModifyResponse: func(resp *http.Response) error {
			log.Println("resp status:", resp.Status)
			log.Println("resp headers:")
			for hk, hv := range resp.Header {
				log.Println(hk, ":", strings.Join(hv, ","))
			}
			return nil
		},
		ErrorLog: log.New(logger.Runtime, "ReverseProxy:", log.LstdFlags|log.Lshortfile),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if err != nil {
				log.Println("ErrorHandler catch err:", err)

				w.WriteHeader(http.StatusBadGateway)
				_, _ = fmt.Fprintf(w, err.Error())
			}
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   60 * time.Second, //连接超时
				KeepAlive: 60 * time.Second, //长连接超时时间
			}).DialContext,
			MaxIdleConns:          100,              //最大空闲连接
			IdleConnTimeout:       90 * time.Second, //空闲超时时间
			TLSHandshakeTimeout:   10 * time.Second, //tls握手超时时间
			ExpectContinueTimeout: 1 * time.Second,  //100-continue 超时时间
		},
	}
	return proxy
}
