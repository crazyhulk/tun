package tunnel

import (
	"encoding/binary"
	"fmt"
	"log"
	"main/packet"
	"net"
	"runtime"
	"time"

	"github.com/songgao/water"
)

func (m *Manager) StartListenUDP() {
	if m.Port == "" {
		log.Printf("invalid port")
		return
	}
	laddr, err := net.ResolveUDPAddr("udp", ":"+m.UDPPort)
	if err != nil {
		return
	}
	fmt.Println(laddr)
	ln, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return
	}

	var packets = make(packet.Packet, 65535)
	for {
		packets.Resize(65535)
		count, addr, err := ln.ReadFromUDP(packets)
		if err != nil {
			log.Printf("accept udp failed. %s\n", err)
			continue
		}

		if info, ok := tunPool[addr.IP.String()]; ok {
			flag := binary.LittleEndian.Uint32(packets[0:3])
			if flag >= SENDIP {
				// 这里处理特殊 datagram 协商
				return
			}

			fmt.Println("======packets:", packets)
			n, err := info.Tun.Write(packets[0:count])
			fmt.Println("======write:", n, err)
			return
		}

		tun, err := initTunInterface()
		if err != nil {
			fmt.Println("initTunInterface:", err)
			continue
		}

		hip, cip := allocIPByTunName(tun.Name())
		headerBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(headerBuf, SENDIP)

		tunPool[addr.String()] = TunInfo{
			Tun:  tun,
			Time: time.Now(),
		}

		data := append(headerBuf, ([]byte)(hip[12:16])...)
		data = append(data, ([]byte)(cip[12:16])...)
		ln.WriteToUDP(data, addr)

		go tunToUDP(ln, addr, tun)
	}

}

func tunToUDP(conn *net.UDPConn, addr *net.UDPAddr, tun *water.Interface) error {
	// 此处大小应该不大于协商的 mtu 这里默认用的1500
	var packets = make(packet.Packet, 65535)
	//	var headerBuf = make([]byte, 4)
	for {
		packets.Resize(65535)
		n, err := tun.Read(packets)
		if err != nil {
			log.Fatal(err)
			return err
		}
		packets = packets[:n]
		// 写入全局 socket
		if conn == nil {
			fmt.Printf("Conn is null, ignore.\n")
			time.Sleep(time.Second)
			continue
		}

		count, err := conn.WriteToUDP(packets, addr)
		if err != nil {
			releaseByTunName(tun.Name())
			tun.Close()
			conn = nil
			tun = nil
			log.Printf("write left body:%d to socket failed. %s\n", count, err)
			runtime.Goexit()
			return err
		}
	}

	return nil
}
