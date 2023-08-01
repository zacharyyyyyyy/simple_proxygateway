# simple_proxygateway
## 简易服务发现代理网关

* 基于etcd服务发现，利用go-cache做本地缓存

````
etcd
key : 对应服务名称
val传输结构 :
{
  [
     "url":"http://127.0.0.1:80",
     "weight":0
  ]
}
````
* 基于httputil.ReverseProxy作url转发，提供ip hash,随机，轮询及权重四种负载均衡模式
* 目前默认path第一位为对应转发服务，即127.0.0.1:8080/test/get?val=1  test为对应服务
* 目前提供黑名单&限流中间件

### 文件结构
<details>
<pre><code>
├── main.go 程序入口
│
├── logger  本地日志记录相关
│
├── config  配置模型相关
│
├── etcd  基于etcd服务发现等逻辑
│
├── transmit  转发部分逻辑
│     └── middleware 转发中间件
│
└── output  日志输出相关
</code></pre>
</details>