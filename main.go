package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"main/packet"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/go-errors/errors"
	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"
)

type AppConfig struct {
	// 角色，取值为 server，client
	Role string `json:"role"`
	// 服务端公网地址
	ServerAddr string `json:"server_addr"`
	// 本机监听端口，服务端使用
	ListenAddr string `json:"listen_addr"`
	// 本机 TUN 设备网段, 192.168.1.100/24
	TunAddr string `json:"tun_addr"`
}

// 两个全局变量，用于交换数据，如果要更好的使用，需要重构此处
var gl_conn *net.TCPConn
var gl_ifce *water.Interface

func LoadConfig(filename string) (cfg *AppConfig, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	var appcfg AppConfig
	err = json.Unmarshal(data, &appcfg)
	if err != nil {
		return
	}

	cfg = &appcfg
	return
}

/*
main 流程
1，加载配置
2，打开 TUN 设备
3，配置 TUN 设备 IP 地址
4，执行服务器或客户端逻辑
*/
func main() {
	go httpServer()
	cfgfile := "./vpn.json"
	appcfg, err := LoadConfig(cfgfile)
	if err != nil {
		log.Fatalf("read %s failed, reason: %s\n", cfgfile, err)
	}

	tuncfg := water.Config{
		DeviceType: water.TUN,
	}

	gl_ifce, err = water.New(tuncfg)

	fmt.Println(gl_ifce.Name())
	if err != nil {
		log.Fatal(err)
	}

	args := []string{gl_ifce.Name(), "10.0.0.1", "pointopoint", "10.0.0.2", "up", "mtu", "1500"}
	if err = exec.Command("/sbin/ifconfig", args...).Run(); err != nil {
		fmt.Println("error")
		return
	}

	if err != nil {
		log.Fatalf("link by name tunnel failed. %s\n", err)
	}
	//addr, err := ifce.Addrs()
	if err != nil {
		log.Fatalf("parse cidr %s failed. %s\n", appcfg.TunAddr, err)
	}

	if appcfg.Role == "server" {
		log.Printf("server mode\n")
		go serverListen(appcfg)
	} else if appcfg.Role == "client" {
		log.Printf("client mode\n")
		go dialServer(appcfg)
	} else {
		log.Fatalf("unknown role. %s", appcfg.Role)
	}

	//var packets = make([]byte, 65535)
	var packets = make(packet.Packet, 65535)
	var headerBuf = make([]byte, 4)
	for {
		packets.Resize(65535)
		n, err := gl_ifce.Read(packets)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
			return
		}
		packets = packets[:n]
		// 写入全局 socket
		if gl_conn == nil {
			fmt.Printf("Conn is null, ignore.\n")
			time.Sleep(time.Second)
			continue
		}
		fmt.Printf("read tun count:%d, is_ipv4:%v \n", n, waterutil.IsIPv4(packets))
		fmt.Println("des:", waterutil.IPv4Destination(packets))
		fmt.Println("source:", waterutil.IPv4Source(packets))

		//count, err := io.Copy(gl_conn, gl_ifce)

		binary.LittleEndian.PutUint32(headerBuf, uint32(n))
		fmt.Println(headerBuf)
		count, err := gl_conn.Write(headerBuf)
		if err != nil {
			log.Printf("write left body to socket failed. %s\n", err)
			continue
		}

		count, err = gl_conn.Write(packets)
		fmt.Println("write conn:", count)
		if err != nil {
			log.Printf("write left body to socket failed. %s\n", err)
			continue
		}
	}
}

func serverListen(cfg *AppConfig) (err error) {
	laddr, err := net.ResolveTCPAddr("tcp", cfg.ListenAddr)
	if err != nil {
		return
	}
	fmt.Println(laddr)
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return
	}

	for {
		fmt.Println("get new client")
		client, err := ln.AcceptTCP()
		fmt.Printf("geted new client: %-v", client)
		if err != nil {
			log.Printf("accept tcp failed. %s\n", err)
			continue
		}

		log.Printf("recv client connection.\n")

		//err = handleConnNew(client)
		err = handleConn(client)
	}
}

func dialServer(cfg *AppConfig) (err error) {
	saddr, err := net.ResolveTCPAddr("tcp4", cfg.ServerAddr)
	if err != nil {
		log.Fatalf("resolve tcp addr %s", err)
	}

	conn, err := net.DialTCP("tcp4", nil, saddr)
	if err != nil {
		log.Fatalf("dial to %s failed. %s\n", saddr.String(), err)
	}
	log.Printf("connected to server.\n")

	return handleConnNew(conn)
	//return handleConn(conn)
}

func handleConnNew(conn *net.TCPConn) error {
	gl_conn = conn
	conn_r := bufio.NewReader(conn)
	if_w := bufio.NewWriter(gl_ifce)

	defer func() {
		conn.Close()
		fmt.Println("conn close")
		gl_conn = nil
	}()

	var bufPool = make([]byte, 1500)
	for {
		n, err := conn_r.Read(bufPool)
		fmt.Println("read bufpool:", n, "byte")
		if err != nil {
			fmt.Println("read failed:", n, err)
			return err
		}
		validBuf := bufPool[:n]
		fmt.Println("read from iOS:", validBuf)
		n, aErr := if_w.Write(validBuf)
		//n, aErr := gl_ifce.Write(validBuf)
		if_w.Flush()

		fmt.Println("write tun:", n, "byte")
		if aErr != nil {
			fmt.Println(n, errors.Wrap(aErr, 0))
			fmt.Println("avalivable:", if_w.Available())
		}
	}

}

func handleConn(conn *net.TCPConn) (err error) {
	gl_conn = conn
	defer func() {
		conn.Close()
		fmt.Println("conn close")
		gl_conn = nil
	}()

	var headerCount = make([]byte, 4)
	for {
		_, err = io.ReadFull(conn, headerCount)
		if err != nil {
			log.Printf("read failed %s\n", err)
			return err
		}
		fmt.Println("header", headerCount)
		count := binary.LittleEndian.Uint32(headerCount)
		fmt.Println("received :", count)
		var bufPool = make([]byte, count)
		_, err = io.ReadFull(conn, bufPool)

		fmt.Println("des:", waterutil.IPv4Destination(bufPool))
		fmt.Println("source:", waterutil.IPv4Source(bufPool))

		if err != nil {
			log.Printf("read failed %s\n", err)
			return
		}

		n, err := gl_ifce.Write(bufPool)
		if err != nil {
			fmt.Println(n, count, err)
			return err
		} else {
			fmt.Println("write tun:", n)
		}
	}

	return
}

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

	tuncfg := water.Config{
		DeviceType: water.TUN,
	}

	gl_ifce, err = water.New(tuncfg)

	fmt.Println(gl_ifce.Name())
	if err != nil {
		log.Fatal(err)
	}

	args := []string{tuncfg.Name, "10.0.0.1", "pointopoint", "10.0.0.2", "up", "mtu", "1500"}
	if err = exec.Command("/sbin/ifconfig", args...).Run(); err != nil {
		fmt.Println("error")
		return
	}

	//// Step 2
	//server, err := net.Dial("tcp", host)
	//if err != nil {
	//	return
	//}
	//client.Write([]byte("HTTP/1.0 200 Connection Established\r\n\r\n"))

	// Step 3
	io.Copy(gl_ifce, client)
	go io.Copy(client, gl_ifce)
}
