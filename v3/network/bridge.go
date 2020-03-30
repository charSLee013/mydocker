package network

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"log"
	"net"
	"os/exec"
	"strings"
	"time"
)

type BridgeNetworkDriver struct {
}


func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

func (d *BridgeNetworkDriver) Create(subnet , name string) (*Network,error) {
	ip,ipRange,err := net.ParseCIDR(subnet)
	if err != nil {
		log.Panicf("Parse subnet error %v",err)
	}
	ipRange.IP = ip
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}

	// 设置好路由
	err = d.initBridge(n)
	if err != nil {
		log.Printf("error init bridge: %v",err)
	}

	return n,err
}

func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	// 添加Bridge设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName);err != nil {
		return fmt.Errorf("Errof add bridge: %s error %v",bridgeName,err)
	}

	// 设置网卡IP
	getewayIP := *n.IpRange
	getewayIP.IP = n.IpRange.IP

	if err := setInterfaceIP(bridgeName,getewayIP.String());err != nil {
		return fmt.Errorf("Error set bridge up: %s error: %v",bridgeName,err)
	}

	// 设置防火墙
	if err := setupIPTables(bridgeName,n.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s: %v",bridgeName,err)
	}

	return nil
}

// 设置防火墙做 MASQUERADE 策略
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE",subnet.String(),bridgeName)
	cmd := exec.Command("iptables",strings.Split(iptablesCmd," ")...)
	output,err :=  cmd.Output()
	if err != nil {
		log.Printf("iptables Output %v",output)
	}
	return err
}

func setInterfaceIP(name string,rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	for i:=0;i < retries;i ++ {
		iface,err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Printf("error retrieving new bridge netlink link [ %s ]... retrying",name)
		time.Sleep(2 * time.Second)
	}


	if err != nil {
		return fmt.Errorf("Abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v",err)
	}

	ipNet,err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	add := &netlink.Addr{
		IPNet: ipNet,
		Peer:ipNet,
		Label:"",
		Flags:0,
		Scope:0,
		Broadcast:nil,
	}
	return netlink.AddrAdd(iface,add)
}

func createBridgeInterface(bridgeName string) error {
	// 返回指定名字的网卡信息
	_,err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(),"no such network interface"){
		return err
	}

	// create *netlink.Bridge object
	la := netlink.NewLinkAttrs()

	la.Name = bridgeName

	br := &netlink.Bridge{
		LinkAttrs: la,
	}
	if err := netlink.LinkAdd(br);err != nil {
		return fmt.Errorf("Brigge creatrion failed for bridge %s: %v",bridgeName,err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br,err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

func (d *BridgeNetworkDriver) Connect(network *Network,endpoint *Endpoint) error {
	bridgeName := network.Name
	br,err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	endpoint.Device = netlink.Veth{
		LinkAttrs:        la,
		PeerName:         "cif-"+endpoint.ID[:5],
		PeerHardwareAddr: nil,
	}

	if err = netlink.LinkAdd(&endpoint.Device);err != nil {
		return fmt.Errorf("Error add endpoint Device: %v",err)
	}

	if err = netlink.LinkSetUp(&endpoint.Device);err != nil {
		return fmt.Errorf("Error add endpoint device: %v",err)
	}

	return err
}