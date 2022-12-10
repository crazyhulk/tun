package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"main/packet"
	"net"
	"runtime"
	"time"

	"github.com/songgao/water"
)

func (m *Manager) StartListenTCP() {
	if m.Port == "" {
		log.Printf("invalid port")
		return
	}
	laddr, err := net.ResolveTCPAddr("tcp", ":"+m.Port)
	if err != nil {
		return
	}
	fmt.Println(laddr)
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return
	}

	for {
		client, err := ln.AcceptTCP()
		if err != nil {
			log.Printf("accept tcp failed. %s\n", err)
			continue
		}
		tun, err := initTunInterface()
		if err != nil {
			fmt.Println("initTunInterface:", err)
			continue
		}

		hip, cip := allocIPByTunName(tun.Name())

		err = upTun(tun, hip, cip)
		if err != nil {
			fmt.Println("upTun", err)
			continue
		}
		err = sendIPs(client, hip, cip)
		if err != nil {
			fmt.Println("sendIPs", err)
			continue
		}
		go tunToTcp(client, tun)
		go tcpToTun(client, tun)
	}

}

func tunToTcp(conn *net.TCPConn, tun *water.Interface) (err error) {
	// 此处大小应该不大于协商的 mtu 这里默认用的1500
	var packets = make(packet.Packet, 65535)
	var headerBuf = make([]byte, 4)
	for {
		packets.Resize(65535)
		n, err := tun.Read(packets)
		if err != nil {
			return err
		}
		packets = packets[:n]
		// 写入全局 socket
		if conn == nil {
			fmt.Printf("Conn is null, ignore.\n")
			time.Sleep(time.Second)
			continue
		}
		//		fmt.Printf("read tun count:%d, is_ipv4:%v \n", n, waterutil.IsIPv4(packets))
		//		fmt.Println("des:", waterutil.IPv4Destination(packets))
		//		fmt.Println("source:", waterutil.IPv4Source(packets))

		//count, err := io.Copy(gl_conn, gl_ifce)

		binary.LittleEndian.PutUint32(headerBuf, uint32(n))
		//fmt.Println(headerBuf)
		count, err := conn.Write(headerBuf)
		if err != nil {
			releaseByTunName(tun.Name())
			tun.Close()
			conn = nil
			tun = nil
			log.Printf("write left body:%d to socket failed. %s\n", count, err)
			runtime.Goexit()
			return err
		}

		count, err = conn.Write(packets)
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

	return
}

func tcpToTun(conn *net.TCPConn, tun *water.Interface) (err error) {
	defer func() {
		err := conn.Close()
		fmt.Println("conn close", err)
		conn = nil
	}()

	var headerCount = make([]byte, 4)
	for {
		_, err = io.ReadFull(conn, headerCount)
		if err != nil {
			log.Printf("read failed %s\n", err)
			return err
		}
		//fmt.Println("header", headerCount)
		count := binary.LittleEndian.Uint32(headerCount)
		//fmt.Println("received :", count)
		if count > 1500 {
			fmt.Println("headerbuf: ======", headerCount, count)
			iErr := invalidPacket(conn)
			if iErr != nil {
				return
			}
			continue
			logPool := make([]byte, 1500)
			_, err = io.ReadFull(conn, logPool)
			if err != nil {
				log.Printf("read failed %s\n", err)
				return
			}

			//fmt.Println(headerCount, logPool)
			_, err = tun.Write(logPool)
			if err != nil {
				log.Printf("read failed %s\n", err)
				return
			}

		}
		var bufPool = make([]byte, count)
		_, err = io.ReadFull(conn, bufPool)

		//		fmt.Println("des:", waterutil.IPv4Destination(bufPool))
		//		fmt.Println("source:", waterutil.IPv4Source(bufPool))

		if err != nil {
			log.Printf("read failed %s\n", err)
			return
		}

		n, err := tun.Write(bufPool)
		if err != nil {
			fmt.Println(n, count, err)
			return err
		} else {
			//fmt.Println("write tun:", n)
		}
		fmt.Println("write tun:===============", n, err, bufPool)
	}

	return
}

func sendIPs(conn *net.TCPConn, hostIP, clentIP net.IP) error {
	var headerCount = make([]byte, 4)
	_, err := io.ReadFull(conn, headerCount)
	if err != nil {
		log.Printf("read failed %s\n", err)
		return err
	}
	//fmt.Println("header", headerCount)
	count := binary.LittleEndian.Uint32(headerCount)
	if count != SENDIP {
		return fmt.Errorf("need consult with ip address")
	}

	headerBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(headerBuf, SENDIP)
	fmt.Println("=====", hostIP.String(), clentIP.String())
	_, err = conn.Write(headerBuf)

	_, err = conn.Write(hostIP[12:16])
	_, err = conn.Write(clentIP[12:16])
	if err != nil {
		log.Printf("send ip failed %s\n", err)
		return err
	}
	return nil
}

func invalidPacket(conn *net.TCPConn) error {
	headerBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(headerBuf, 0xFFFFFFFF)
	conn.Write(headerBuf)
	return nil
}
