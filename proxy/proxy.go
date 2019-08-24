package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
)

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Println("hello")
	fmt.Fprintf(w, "hello\n")
}

func HttpServer() {
	//http.HandleFunc("/hello", hello)
	//http.ListenAndServe(":8090", nil)
	proxy := NewProxy()
	http.ListenAndServe("0.0.0.0:8091", proxy)
}

/// Test code followed this line

type Pxy struct{}

func NewProxy() *Pxy {
	return &Pxy{}
}

// ServeHTTP is the main handler for all requests.
func (p *Pxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Printf("Received request %s %s %s\n",
		req.Method,
		req.Host,
		req.RemoteAddr,
	)

	fmt.Println("http.method", req.Method)
	if req.Method != http.MethodConnect {
		return
	}
	// Step 1
	host := req.URL.Host
	println(host)
	hij, ok := rw.(http.Hijacker)
	if !ok {
		panic("HTTP Server does not support hijacking")
	}

	client, _, err := hij.Hijack()
	if err != nil {
		return
	}
	if _, err := client.Write([]byte("hello")); err != nil {
		fmt.Println(err)
		return
	}

	// Step 2
	server, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	client.Write([]byte("HTTP/1.0 200 Connection Established\r\n\r\n"))

	// Step 3
	io.Copy(server, client)
	go io.Copy(client, server)
}
