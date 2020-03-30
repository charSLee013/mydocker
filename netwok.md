## 容器网络

---

### `Linux Veth`

```Bash
## 创建两个network namespace
sudo ip netns add ns1
sudo ip netns add ns2

## 创建一对 veth
sudo ip link add veth0 type veth peer name veth1

## 分别将两个Veth移到两个Namespace中
sudo ip link set veth0 netns ns1
sudo ip link set veth1 netns ns2

## 查看 ns1 的网络设备
sudo ip netns exec ns1 ip link

## 配置每个veth的网络地址和Namespace的路由
sudo ip netns exec ns1 ifconfig veth0 172.18.0.2/24 up
sudo ip netns exec ns2 ifconfig veth1 172.18.0.3/24 up

sudo ip netns exec ns1 route add default dev veth0
sudo ip netns exec ns2 route add default dev veth1

## 通过veth一端出去的包,另外一端能够直接接受到
sudo ip netns exec ns1 ping -c 1 172.18.0.3
```

---

### `Linux Bridge`
Linux Bridge（网桥）是工作于二层的虚拟网络设备，功能类似于物理的交换机。
Bridge可以绑定其他Linux网络设备作为从设备，并将这些设备虚拟化为端口，当一个从设备被绑定到Bridge上时，就相当于真实网络中的交换机端口插入了一个连接有终端的网线

```Bash
## 创建Veth设备并将一端移入Namespace
sudo ip netns add ns1
sudo ip link add veth0 type veth peer name veth1
sudo ip link set veth1 netns ns1

## 创建网桥
sudo brctl addbr br0

## 挂载网络设备
sudo brctl addif br0 eth0
sudo brctl addif br0 veth0
```

---

### `Linux`路由表
通过路由表来决定在某个网络`Namespace`中的包流向,从而定义请求会到哪个网络设备上

```Bash
## 启动虚拟网络设备,并设置它在Net namespace 中的IP地址
sudo ip link set veth0 up
sudo ip link set br0 up
sudo ip netns exec ns1 ifconfig veth1 172.18.0.2/24 up

## 分别设置ns1网络设备的路由和宿主机上的路由
## default 代表 0.0.0.0/0,即在Net Namespace 中所有流量都经过veth1的网络设备流出
sudo ip netns exec ns1 route add default dev veth1

## 在宿主机上将 172.18.0.0/24 的网段请求路由到br0的网桥
sudo route add -net 172.18.0.0/24 dev br0
```

通过设置路由,对`IP`地址的请求就能正确被路由到对应的网络设备上,从而实现通信

```bash
## 查看宿主机的IP地址
sudo ifconfig eth0

## 从Namespace中访问宿主机的地址
sudo ip netns exec ns1 ping -c 1 172.22.11.211 
```

---

### `Linux iptables`
iptables 是一个配置 Linux 内核 防火墙 的命令行工具，是 netfilter 项目的一部分。术语 iptables 也经常代指该内核级防火墙。iptables 可以直接配置，也可以通过许多 前端[broken link: invalid section] 和 图形界面[broken link: invalid section] 配置。iptables 用于 ipv4，ip6tables 用于 ipv6

#### `MASQUERADE` IP伪装(NAT)

`MASQUERADE`策略可以将请求包中的源地址转换成一个网络设备的地址
比如`Namespace`中的网络设备的地址是`172.18.0.2`,这个地址虽然在宿主机上可以路由到`br0`的网桥,但是到达宿主机的外部之后,是不知道如何路由到这个`IP`地址的,所以如果请求外部地址的话,需要先通过`MASQUERADE`策略将这个`IP`转换成宿主机出口网卡的`IP`

```Bash
## 打开IP转发
sudo sysctl -w net.ipv4.conf.all.forwarding=1
## 对Namespace中发出的包添加网络地址转换
sudo iptables -t nat -A POSTROUTING -s 172.18.0.0/24 -o eth0 -j MASQUERADE
```

在`Namespace`中请求宿主机外部地址时,将`Namespace`中的源地址转换成宿主机的地址作为源地址,就可以在`Namespace`中访问宿主机

#### `DNAT` 
（Destination Network Address Translation,目的地址转换) 通常被叫做目的映谢,经常用于将内部网络地址的端口映射到外部去,比如在`Namespace`里开了`80`端口要映射到宿主机的`80`端口,这时可以用`DNAT`策略

```Bash
## 将宿主机上80端口的请求转发到Namespace的IP上
sudo iptables -t nat -A PREROUTING -p tcp -m tcp --dport 80 -j DNAT --to-destination 172.18.0.2:80
```

---

### 用`Go`配置容器网络


#### `net`库

* `net`库是`Go`语言的内置库,提供了跨平台支持的网络地址处理,以及各种常见的网络协议诸如`TCP`,`UDP`,`DNS`,`Unix Socket`等

```Go
import "net"

net.IP  //这个类型定义了IP地址的数据结构,并通过ParseIP和String方法将字符串与其转换
net.IPNet // 这个类型定义了IP段的数据结构,比如 192.168.0.0/16这样的网段,同样可以通过ParseCIDR和String方法与字符串转换
```

* `github.com/vishvananda/netlink`
是Go语言的操作网络接口,路由表等配置的库,使用它的调用可以方便我们通过`IP`命令去管理网络接口

* `github.com/vishvananda/netns`
通过这个库可以将当前代码执行的进程加入指定的`Net Namespace`中

---

### 构建网络模型

网络是容器的一个集合,在这个网络上的容器可以通过这个网络互相通信,就像挂载到同一个`Linux Bridge`设备上的网络设备一样,可以直接通过`Bridge`设备实现网络互连;连接到同一个网络中的容器也可以通过这个网络和网络中别的容器互连,网络中会包括这个网络相关的配置,比如网络的容器地址段,网络操作所调用的网络驱动等信息.

```Go
type Network struct {
    Name string  // 网络名
    IpRange *net.IPNet // 地址段
    Driver string   // 网络驱动名
}
```

#### 网络端点

网络端点用于连接容器与网络的,保证容器内部与网络的通信,像之前提过的`Veth`,一端挂载到容器内部,另一端挂载挂载到`Bridge`上,就能保证容器和网络的通信.
网络端点中会包括连接到网络的一些信息,比如地址,`Veth`设备,端口映射,连接的容器和网络等信息

```Go
type Endpoint struct {
    ID string `json:"id"`
    Device netlink.Veth `json:"dev"`
    IPAddress net.IP `json:"ip"`
    MacAddress net.HardwareAddr `json:"mac"`
    PortMapping []string `json:"postmapping"`
    Network *Network
}
```

---

而网络端点的信息传输需要靠网络功能的两个组件配合完成,这个两个组件分别为网络驱动和`IPAM`,我们还需要

#### 网络驱动

网络驱动(Network driver)是一个网络功能中的组件,不同的驱动对网络的创建,连接,销毁的策略不同,通过在创建网络时指定不同的网络驱动来定义使用哪个驱动做网络的配置,定义如下:

```go
type NetworkDriver interface {
    // 驱动名
    Name() string
    // 创建网络
    Create(subnet string,name string) (*Network,error)
    // 删除网络
    Delete(network Network) error
    // 连接容器网络端点
    Connect(network *Network, endpoint *Endpoint) error
    // 从网络上移除容器网络端点
    Disconnet(network Network, endpoint *Endpoint) error
}
```

#### IPAM
IPAM（IP Address Management） 用户发现,监视,审核和管理网络IP地址,主要用于网络IP地址的分配和释放,包括容器的IP地址和网络网关的IP地址,主要功能如下:
* `IPAM.Allocate(subnet *net.IPNet)`从指定的`subnet`网段中分配`IP`地址
* `IPAM.release(subnet net.IPNet, ipaddr net.IP)`从指定的`subnet`网段中释放掉指定的`IP`地址


#### 调用关系

##### 创建网络
* 通过`Bridge`的网络驱动创建一个网络,网段是`192.168.0.0/24`,网络驱动是`Bridge`,步骤如下
1. 调用指定的网络驱动创建网络
2. 调用`IPAM`获取IP段和GatewayIR
3. 配置网络设备