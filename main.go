package main

import (
	"main/proxy"
	"main/tunnel"
)

/*
main 流程
1，加载配置
2，打开 TUN 设备
3，配置 TUN 设备 IP 地址
4，执行服务器或客户端逻辑
*/
func main() {
	go proxy.HttpServer()
	manager := tunnel.Manager{}
	manager.Port = "8080"
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
