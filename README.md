# Pyrite

**pyrite-go** 是 Pyrite 协议的 Go 实现。

### 安装 & 引入

使用 `go get github.com/mo-crystal/pyrite-go@latest` 将 Pyrite 导入你的项目。

使用 `import pyrite "github.com/mo-crystal/pyrite-go"` 在你的项目中使用 Pyrite。

### 用法示例

```go
addr := net.UDPAddr{
	Port: 648,
	IP:   net.ParseIP("127.0.0.1"),
}

// 创建一个客户端
// 第一个参数为服务器地址，第二个参数为超时时间
// 创建时会尝试连接服务器。若连接成功则返回客户端指针和 nil 错误
client, err := pyrite.NewClient(addr, 10*time.Second)
if err != nil {
	// 端口占用等因素可能导致 Client 创建失败
	panic(err)
}

// 根据 identifier，为来自服务器的数据包选择特定的处理函数
// 如果此函数返回空字符串，Pyrite 就认为此请求不需要响应，则不会向服务器回复 ACK
client.AddRouter("process-data", func(req string) string {
	return "hello this is server"
})

// client.Start 会启动客户端的主循环
// 此主循环会接受所有来自服务器的数据，并识别其 identifier
// 依据此前 client.AddRouter 的逻辑，自动分配处理函数
// 此函数收到自服务器发来的 ACK 报文时，会将其分发给特定的 Promise，并解除其阻塞
go client.Start()

// 使用 client.Promise 可以向服务器发送一条消息，并让服务器用特定的函数处理这条消息
// client.Promise 会阻塞当前线程，直到服务器响应此 Promise，或者超时
// 服务器会对 client.Promise 发送的数据进行回复，回复的内容为第一个返回值
resp, err := client.Promise("please-check-this", "hi! this is client!")
if err != nil {
	// 若网络环境等因素导致丢包，client.Promise 在超时时间后会将错误信息返回
	panic(err)
}

// 使用 client.Tell 可以向服务器发送一条消息，并让服务器用特定的函数处理这条消息
// client.Tell 不期望服务器对此回信。服务器处读取此消息后应当不会有任何动作
err = client.Tell("check-this, but dont response", "hi! this is client!")
if err != nil {
	// 虽然不期望对方返回信息，但此函数仍然会检查数据长度
	// 数据过长时，会抛出错误
	panic(err)
}
```
