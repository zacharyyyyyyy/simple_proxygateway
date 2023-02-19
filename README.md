# simple_proxygateway
## 简易服务发现代理网关

* 基于etcd服务发现，利用go-cache做本地缓存

````
etcd传输结构
{
  [
     "url":127.0.0.1,
     "weight":0
  ]
}
````
* 基于httputil.ReverseProxy作url转发，提供ip hash,随机，轮询及权重四种负载均衡模式

###文件结构
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
│
└── output  日志输出相关
</code></pre>
</details>