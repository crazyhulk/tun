package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
	//"github.com/vishvananda/netlink"
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

func gzipEncode(in []byte) (out []byte, err error) {
	var buffer bytes.Buffer
	gzipW := gzip.NewWriter(&buffer)
	defer gzipW.Close()

	_, err = gzipW.Write(in)
	if err != nil {
		gzipW.Close()
		return
	}

	err = gzipW.Close()
	if err != nil {
		return
	}

	out = buffer.Bytes()
	return
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
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

	tuncfg.Name = "tunnel"
	gl_ifce, err = water.New(tuncfg)
	
	fmt.Println(gl_ifce.Name())
	//ifce, err := net.InterfaceByName("tun0")
	if err != nil {
		log.Fatal(err)
	}

	// 改 IP
	//tunnel, err := netlink.LinkByName("tunnel")
	if err != nil {
		log.Fatalf("link by name tunnel failed. %s\n", err)
	}
	//addr, err := netlink.ParseAddr(appcfg.TunAddr)
	//addr, err := ifce.Addrs()
	if err != nil {
		log.Fatalf("parse cidr %s failed. %s\n", appcfg.TunAddr, err)
	}

	//netlink.AddrAdd(tunnel, addr)
	//netlink.LinkSetUp(tunnel)

	if appcfg.Role == "server" {
		log.Printf("server mode\n")
		go serverListen(appcfg)
	} else if appcfg.Role == "client" {
		log.Printf("client mode\n")
		go dialServer(appcfg)
	} else {
		log.Fatalf("unknown role. %s", appcfg.Role)
	}

	var frame ethernet.Frame
	for {
		frame.Resize(1500)
		fmt.Println("start read tun")
		n, err := gl_ifce.Read([]byte(frame))
		fmt.Println(n)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			continue
		}
		frame = frame[:n]
		fmt.Println("Read from tun & write to conn")
		fmt.Println(frame[:min(n, 20)])
		// 写入全局 socket
		if gl_conn == nil {
			fmt.Printf("Conn is null, ignore.\n")
			continue
		}

		n, err = gl_conn.Write([]byte(frame))
		if err != nil {
			log.Printf("write left body to socket failed. %s\n", err)
			continue
		}
	}
}

func serverListen(cfg *AppConfig) (err error) {
	laddr, err := net.ResolveTCPAddr("tcp4", cfg.ListenAddr)
	if err != nil {
		return
	}
	fmt.Println(laddr)
	ln, err := net.ListenTCP("tcp4", laddr)
	if err != nil {
		return
	}

	for {
		client, err := ln.AcceptTCP()
		if err != nil {
			log.Printf("accept tcp failed. %s\n", err)
			time.Sleep(time.Second)
			continue
		}

		log.Printf("recv client connection.\n")
		handleConn(client)
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

	return handleConn(conn)
}

func handleConn(conn *net.TCPConn) (err error) {
	gl_conn = conn
	defer func() {
		conn.Close()
		gl_conn = nil
	}()

	bufPool := make([]byte, 1500)
	for {
		count, aErr := io.ReadFull(conn, bufPool)
		//_, err = io.ReadFull(conn, headBuf)
		fmt.Printf("read from conn , count: %d \n", count)
		validBuf := bufPool[:count]
		fmt.Println(validBuf[:min(count, 20)])
		if aErr != nil {
			log.Printf("read head failed %s\n", err)
			return
		}
		n, aErr := gl_ifce.Write(validBuf)
		fmt.Println("write count: %d\n",n)
		fmt.Println(aErr)
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
}