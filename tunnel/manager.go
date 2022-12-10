package tunnel

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"

	"github.com/songgao/water"
)

const (
	SENDIP = uint32(0x00010000)

	connecting = int32(1)
	connected  = int32(2)
)

var tunTable = map[string]net.IP{}

var managerPool = map[string]Manager{}

var tunPool = map[string]TunInfo{}

type TunInfo struct {
	Tun  *water.Interface
	Time time.Time
}

type Manager struct {
	Host  string
	Port  string
	State int32

	UDPPort string
}

func (m *Manager) Start() {
	go m.StartListenTCP()
	m.StartListenUDP()
}

func initTunInterface() (tun *water.Interface, err error) {
	tuncfg := water.Config{
		DeviceType: water.TUN,
	}
	tun, err = water.New(tuncfg)

	fmt.Println(tun.Name())
	if err != nil {
		panic(err)
	}
	return tun, nil
}

func upTun(tun *water.Interface, hostIP, clentIP net.IP) (err error) {
	args := []string{tun.Name(), hostIP.String(), "pointopoint", clentIP.String(), "up", "mtu", "1500"}
	// mac os 不需要我们指定
	if runtime.GOOS == "darwin" {
		args = []string{tun.Name(), hostIP.String(), clentIP.String(), "up", "mtu", "1500"}
	}

	if err = exec.Command("/sbin/ifconfig", args...).Run(); err != nil {
		fmt.Println("error: can not link up:", tun.Name(), "err:", err)
		panic(err)
		return err
	}
	return
}
