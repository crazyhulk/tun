package main

import (
	"fmt"
	"main/tunnel"
	"net/http"
)

/*
main 流程
1，加载配置
2，打开 TUN 设备
3，配置 TUN 设备 IP 地址
4，执行服务器或客户端逻辑
*/
func main() {
	//httpServer()
	manager := tunnel.Manager{}
	manager.Port = "8080"
	manager.UDPPort = "8081"
	manager.Start()
}

//func dialServer(cfg *AppConfig) (err error) {
//	saddr, err := net.ResolveTCPAddr("tcp4", cfg.ServerAddr)
//	if err != nil {
//		log.Fatalf("resolve tcp addr %s", err)
//	}
//
//	conn, err := net.DialTCP("tcp4", nil, saddr)
//	if err != nil {
//		log.Fatalf("dial to %s failed. %s\n", saddr.String(), err)
//	}
//	log.Printf("connected to server.\n")
//
//	return handleConnNew(conn)
//	//return handleConn(conn)
//}

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Println("hello")
	fmt.Fprintf(w, "hello\n")
}

func httpServer() {
	http.HandleFunc("/hello", hello)
	http.ListenAndServe(":8090", nil)
	//proxy := NewProxy()
	//http.ListenAndServe("0.0.0.0:8091", proxy)
}

/// Test code followed this line

type Pxy struct{}

func NewProxy() *Pxy {
	return &Pxy{}
}

// ServeHTTP is the main handler for all requests.
//func (p *Pxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
//	fmt.Printf("Received request %s %s %s\n",
//		req.Method,
//		req.Host,
//		req.RemoteAddr,
//	)
//
//	fmt.Println("http.method", req.Method)
//	if req.Method != http.MethodConnect {
//		return
//	}
//	// Step 1
//	host := req.URL.Host
//	println(host)
//	hij, ok := rw.(http.Hijacker)
//	if !ok {
//		panic("HTTP Server does not support hijacking")
//	}
//
//	client, _, err := hij.Hijack()
//	if err != nil {
//		return
//	}
//	if _, err := client.Write([]byte("hello")); err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	tuncfg := water.Config{
//		DeviceType: water.TUN,
//	}
//
//	//	gl_ifce, err = water.New(tuncfg)
//	//
//	//	fmt.Println(gl_ifce.Name())
//	//	if err != nil {
//	//		log.Fatal(err)
//	//	}
//
//	args := []string{tuncfg.Name, "10.0.0.1", "pointopoint", "10.0.0.2", "up", "mtu", "1500"}
//	if err = exec.Command("/sbin/ifconfig", args...).Run(); err != nil {
//		fmt.Println("error")
//		return
//	}
//
//	//// Step 2
//	//server, err := net.Dial("tcp", host)
//	//if err != nil {
//	//	return
//	//}
//	//client.Write([]byte("HTTP/1.0 200 Connection Established\r\n\r\n"))
//
//	// Step 3
//	io.Copy(gl_ifce, client)
//	go io.Copy(client, gl_ifce)
//}
