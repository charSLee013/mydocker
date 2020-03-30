package network

import (
	"encoding/json"
	"fmt"
	"github.com/charSLee013/mydocker/v3/driver"
	"github.com/vishvananda/netlink"
	"log"
	"net"
	"os"
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
	if err = netlink.
}