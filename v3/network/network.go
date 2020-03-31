package network

import (
	"encoding/json"
	"fmt"
	"github.com/charSLee013/mydocker/v3/driver"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/gocker/network/network/"
	drivers = map[string]*BridgeNetworkDriver{}
	networks = map[string]*Network{}
)


type Network struct {
	Name string
	IpRange *net.IPNet
	Driver string
}

type NetworkDriver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(network Network) error
	Connect(network *Network, endpoint *Endpoint) error
	Disconnect(network Network, endpoint *Endpoint) error
}

type Endpoint struct {
	ID string `json:"id"`
	Device netlink.Veth `json:"dev"`
	IPAddress net.IP `json:"ip"`
	MacAddress net.HardwareAddr `json:"mac"`
	Network    *Network
	PortMapping []string
}

func Init() error {
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	if _,err := os.Stat(defaultNetworkPath);os.IsNotExist(err) {
		if err = os.MkdirAll(defaultNetworkPath,0644);err != nil {
			return err
		}
	}

	filepath.Walk(defaultNetworkPath, func(nwpath string, info os.FileInfo, err error) error {
		if strings.HasSuffix(nwpath,"/"){
			return nil
		}

		_,nwName := path.Split(nwpath)
		nw := &Network{
			Name:nwName,
		}

		if err := nw.load(nwpath);err != nil {
			log.Printf("Error load network: %v",err)
		}

		networks[nwName] = nw
		return nil
	})

	return nil
}

func (nw *Network) load(dumpPath string) error {
	nwConfigFile,err := os.Open(dumpPath)
	defer nwConfigFile.Close()

	if err != nil {
		return err
	}

	nwJson := make([]byte,2000)
	n,err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(nwJson[:n],nw)
	if err != nil {
		log.Print("Error load nw info",err)
		return err
	}
	return nil
}

func(nw *Network) dump(dumpPath string) error {
	if _,err := os.Stat(dumpPath);os.IsNotExist(err){
		if err = os.MkdirAll(dumpPath,0644);err != nil {
			return err
		}
	}

	nwPath := path.Join(dumpPath,nw.Name)
	nwFile,err := os.OpenFile(nwPath,os.O_TRUNC | os.O_WRONLY | os.O_CREATE,0644)
	if err != nil {
		log.Printf("Error: %v",err)
		return err
	}
	defer nwFile.Close()

	nsJson,err := json.Marshal(nw)
	if err != nil {
		log.Printf("Error: %v",err)
		return err
	}

	_,err = nwFile.Write(nsJson)
	if err != nil {
		log.Printf("Error: %v",err)
		return err
	}

	return nil
}

func(nw *Network) remove(dumpPath string) error {
	if _,err := os.Stat(path.Join(dumpPath,nw.Name)); err!=nil {
		if os.IsNotExist(err){
			return nil
 		} else {
 			return err
		}
	} else {
		return os.Remove(path.Join(dumpPath,nw.Name))
	}
}

func CreateNetwork(driver, subnet,name string) error {
	_,cidr,err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}

	ip,err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}

	cidr.IP = ip

	nw,err := drivers[driver].Create(cidr.String(),name)
	if err != nil {
		return err
	}

	return nw.dump(defaultNetworkPath)
}


func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout,12,1,3,' ',0)
	fmt.Fprint(w,"NAME\tIpRange\tDriver\n")
	for _,nw := range networks{
		fmt.Fprintf(w,"%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver)
	}

	if err := w.Flush();err != nil {
		log.Printf("Flush error %v",err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	nw,ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network : %s",networkName)
	}

	if err:= ipAllocator.Release(nw.IpRange,&nw.IpRange.IP);err != nil {
		return fmt.Errorf("Error Remove Network geteway ip : %s",err)
	}

	if err := drivers[nw.Driver].Delete(*nw);err != nil {
		return fmt.Errorf("Error Remove Network DriverError: %v",err)
	}

	return nw.remove(defaultNetworkPath)
}

func enterContainerNetns(enLink *netlink.Link, cinfo *driver.ContainerInfo) func() {
	f,err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net",cinfo.Pid),os.O_RDONLY,0)
	if err != nil {
		Sugar.Errorf("error get container net namespace %v",err)
	}

	nsFD := f.Fd()
	runtime.LockOSThread()

	// 修改veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink,int(nsFD));err != nil {
		Sugar.Errorf("Error set link netns %v",err)
	}

	// 获取当前网络namespace
	origns,err := netns.Get()
	if err != nil {
		Sugar.Errorf("Error get current netns %v",err)
	}

	// 设置当前进程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD));err != nil {
		Sugar.Errorf("Error set netns %v",err)
	}

	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}


func configEndpointIpAddressAndRoute(ep *Endpoint,cinfo *driver.ContainerInfo) error {
	peerLink,err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v",err)
	}

	defer enterContainerNetns(&peerLink,cinfo)()

	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	if err = setInterfaceIP(ep.Device.PeerName,interfaceIP.String());err != nil {
		return fmt.Errorf("%v,%s",ep.Network,err)
	}

	if err = setInterfaceUP("lo");err != nil {
		return err
	}

	_,cidr,err := net.ParseCIDR("0.0.0.0/0")

	defaultRoute := &netlink.Route{
		LinkIndex:peerLink.Attrs().Index,
		Gw: ep.Network.IpRange.IP,
		Dst:cidr,
	}

	if err = netlink.RouteAdd(defaultRoute);err != nil {
		return err
	}

	return nil
}

func configPortMapping(ep *Endpoint,cinfo *driver.ContainerInfo)error {
	for _,pm := range ep.PortMapping{
		portMapping := strings.Split(pm,":")
		if len(portMapping) != 2{
			Sugar.Errorf("port mapping format error %v",pm)
			continue
		}

		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0],
			ep.IPAddress.String(),
			portMapping[1])
		cmd := exec.Command("iptables",strings.Split(iptablesCmd," ")...)

		output,err := cmd.Output()
		if err != nil {
			Sugar.Errorf("iptbles Output %v",output)
			continue
		}
	}

	return nil
}

func Connect(networkName string,cinfo *driver.ContainerInfo) error {
	network,ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No such Network: %s",networkName)
	}

	// 分配容器IP地址
	ip,err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	// 创建网络端点
	ep := &Endpoint{
		ID: fmt.Sprintf("%s-%s",cinfo.Id,networkName),
		IPAddress:ip,
		Network:network,
		PortMapping:cinfo.PortMapping,
	}

	// 调用网络驱动挂载和配置网络端点
	if err = drivers[network.Driver].Connect(network,ep);err != nil {
		return err
	}

	// 到容器的namespace配置容器网络设备IP地址
	if err = configEndpointIpAddressAndRoute(ep,cinfo);err != nil {
		return err
	}

	return configPortMapping(ep,cinfo)
}

func Disconnect(networkName string,cinfo *driver.ContainerInfo) error {
	return nil
}