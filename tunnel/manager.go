package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"main/packet"
	"net"
	"os/exec"
	"runtime"
	"time"

	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"
)

type Manager struct {
	Host string
	Port string
}

func (m *Manager) Start() {
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
			log.Printf("accept tcp failed. %s\n", err)
			continue
		}

		go tunToTcp(client, tun)
		go tcpToTun(client, tun)
	}

}

func initTunInterface() (tun *water.Interface, err error) {
	tuncfg := water.Config{
		DeviceType: water.TUN,
	}

	tun, err = water.New(tuncfg)

	fmt.Println(tun.Name())
	if err != nil {
		log.Fatal(err)
	}

	args := []string{tun.Name(), "10.0.0.1", "pointopoint", "10.0.0.2", "up", "mtu", "1500"}
	if err = exec.Command("/sbin/ifconfig", args...).Run(); err != nil {
		fmt.Println("error: can not link up:", tun.Name())
		return nil, err
	}

	return tun, nil
}

func tunToTcp(conn *net.TCPConn, tun *water.Interface) (err error) {
	var packets = make(packet.Packet, 65535)
	var headerBuf = make([]byte, 4)
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
		fmt.Printf("read tun count:%d, is_ipv4:%v \n", n, waterutil.IsIPv4(packets))
		fmt.Println("des:", waterutil.IPv4Destination(packets))
		fmt.Println("source:", waterutil.IPv4Source(packets))

		//count, err := io.Copy(gl_conn, gl_ifce)

		binary.LittleEndian.PutUint32(headerBuf, uint32(n))
		fmt.Println(headerBuf)
		count, err := conn.Write(headerBuf)
		if err != nil {
			tun.Close()
			conn = nil
			tun = nil
			log.Printf("write left body to socket failed. %s\n", err)
			runtime.Goexit()
			return err
		}

		count, err = conn.Write(packets)
		fmt.Println("write conn:", count)
		if err != nil {
			tun.Close()
			conn = nil
			tun = nil
			log.Printf("write left body to socket failed. %s\n", err)
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
		fmt.Println("header", headerCount)
		count := binary.LittleEndian.Uint32(headerCount)
		fmt.Println("received :", count)
		if count > 1500 {
			logPool := make([]byte, 1500)
			_, err = io.ReadFull(conn, logPool)
			if err != nil {
				log.Printf("read failed %s\n", err)
				return
			}

			fmt.Println(headerCount, logPool)
			_, err = tun.Write(logPool)
			if err != nil {
				log.Printf("read failed %s\n", err)
				return
			}

		}
		var bufPool = make([]byte, count)
		_, err = io.ReadFull(conn, bufPool)

		fmt.Println("des:", waterutil.IPv4Destination(bufPool))
		fmt.Println("source:", waterutil.IPv4Source(bufPool))

		if err != nil {
			log.Printf("read failed %s\n", err)
			return
		}

		n, err := tun.Write(bufPool)
		if err != nil {
			fmt.Println(n, count, err)
			return err
		} else {
			fmt.Println("write tun:", n)
		}
	}

	return
}
