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
sudo ip netns add ns1   # 创建名为 ns1 的Network Namespace
sudo ip link add veth0 type veth peer name veth1    # veth设备是成对出现的，两个设备之间的数据三相互贯通的
sudo ip link set veth1 netns ns1    # 将veth1 网卡加入到 ns1 的Network Namespace

## 创建网桥
sudo brctl addbr br0

## 挂载网络设备
sudo brctl addif br0 eth0
sudo brctl addif br0 veth0

## 查看挂载情况
sudo brctl show
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
# -t nat: 配置的是nat表, -t 指定了nat表
# -A POSTROUTING: A代表的是append,增加一个 POSTROUTING 的chain
# -p tcp --dport 80 代表的是匹配的数据包特征，使用tcp协议，目的端口号是80
# -j 代表action，DNAT 就是目的地址转化，主要用于外网向内网发送数据包，在PREROUTING链上实现
# --to-destination 172.18.0.2:80: 将报文 IP改成：172.18.0.2 , PORT改成：80，从而实现外网访问内网数据
```

---

### `Linux Veth` 和 `Bridge` 总结

#### 前言
Docker的网络以及Flannel的网络实现都涉及到`Veth`和`Bridge`使用.
在宿主机上创建一个`Bridge`，每到一个容器创建，就会创建一对互通`Veth` (Bridge-veth <--> Container-veth)
一端连接到主机的`Bridge`(docker0),另一端连接到容器的`Network namespace`
可以通过`sudo brctl show`查看`Bridge`连接的`veth`

#### 说明

##### `VETH` (virtual Ethernet)
`Linux Kernel`支持的一种虚拟网络设备，表示一对虚拟的网络接口
`Veth`的两端可以处于不同的`Network namespace`，可以作为主机和容器之间的网络通信
发送到`Veth`一端的请求会从另一端的`Veth`发出

##### `Bridge`
`Bridge` 是 `Linux` 上用来做 `TCP/IP` 二层协议交换的设备，相当于交换机
可以将其他网络设备挂在 `Bridge` 上面
当有数据到达时，`Bridge`会根据报文中的MAC信息进行广播，转发，丢弃.


### 网络拓扑图
```bash
                           +------------------------+
                           |                        | iptables +----------+
                           |  br01 192.168.88.1/24  |          |          |
                +----------+                        <--------->+ eth0   |
                |          +------------------+-----+          |          |
                |                             |                +----------+
           +----+---------+       +-----------+-----+
           |              |       |                 |
           | br-veth01    |       |   br-veth02     |
           +--------------+       +-----------+-----+
                |                             |
+--------+------+-----------+     +-------+---+-------------+
|        |                  |     |       |                 |
|  ns01  |   veth01         |     |  ns02 |  veth01         |
|        |                  |     |       |                 |
|        |   192.168.88.11  |     |       |  192.168.88.12  |
|        |                  |     |       |                 |
|        +------------------+     |       +-----------------+
|                           |     |                         |
|                           |     |                         |
+---------------------------+     +-------------------------+

```
`br01`是创建的`Bridge`，链接着两个`Veth`，两个`Veth`的另一端分别在另外两个`namespace`里
`eth0`是宿主机对外的网卡，`namespace`对外的数据包会通过`SNAT`/`MASQUERADE`出去 


#### 部署`Bridge`和`Veth`

##### 设置`Bridge`

创建`Bridge`

```bash
sudo brctl addbr br01
```

启动`Bridge`

```bash
sudo ip link set dev br01 up
# 也可以用下面这种方式启动
sudo ifconfig br01 up 
```

给`Bridge`分配IP地址

```bash
sudo ifconfig br01 192.168.88.1/24 up
```

##### 创建`Network namespace`

创建两个`namespace`: `ns01` `ns02`

```bash
sudo ip netns add ns01
sudo ip netns add ns02

## 查看创建的ns
sudo ip netns list
ns02
ns01
```

##### 设置`Veth pair`

创建两对`veth`



```bash
# 创建 `VETH` 设备：`ip link add link [DEVICE NAME] type veth`
sudo ip link add veth01 type veth peer name br-veth01
sudo ip link add veth02 type veth peer name br-veth02
```

将其中一端的`Veth`(br-veth$)挂载到`br01`下面

```bash
# attach 设备到 Bridge：brctl addif [BRIDGE NAME] [DEVICE NAME]
sudo brctl addif br01 br-veth01
sudo brctl addif br01 br-veth02

# 查看挂载详情
sudo brctl show br01
bridge name     bridge id               STP enabled     interfaces
br01            8000.321bc3fd56fd       no              br-veth01
                                                        br-veth02
```

启动这两对`Veth`

```bash
sudo ip link set dev br-veth01 up
sudo ip link set dev br-veth02 up
```

将另一端的`veth`分配给创建好的`ns`

```bash
sudo ip link set veth01 netns ns01
sudo ip link set veth02 netns ns02
```

##### 部署`Veth`在`ns`的网络

通过`sudo ip netns [NS] [COMMAND]`命令可以在特定的网络命名空间执行命令

查看`network namespace`里的网络设备:

```bash
sudo ip netns exec ns01 ip addr
1: lo: <LOOPBACK> mtu 65536 qdisc noop state DOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: sit0@NONE: <NOARP> mtu 1480 qdisc noop state DOWN group default qlen 1000
    link/sit 0.0.0.0 brd 0.0.0.0
8: veth01@if7: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN group default qlen 1000
    link/ether d2:88:ec:62:cd:0a brd ff:ff:ff:ff:ff:ff link-netnsid 0
```

可以看到刚刚被加进来的`veth01`还没有IP地址
给两个`network namespace`的`veth`设置IP地址和默认路由
默认网关设置为`Bridge`的`IP`

```bash
sudo ip netns exec ns01 ip link set dev veth01 up
sudo ip netns exec ns01 ifconfig veth01 192.168.88.11/24 up
sudo ip netns exec ns01 ip route add default via 192.168.88.1

sudo ip netns exec ns02 ip link set dev veth02 up
sudo ip netns exec ns02 ifconfig veth02 192.168.88.12/24 up
sudo ip netns exec ns02 ip route add default via 192.168.88.1
```

查看 `ns`的`veth`是否分配了IP

```bash
sudo ip netns exec ns01 ifconfig veth01
sudo ip netns exec ns02 ifconfig veth02

veth02: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 192.168.88.12  netmask 255.255.255.0  broadcast 192.168.88.255
        inet6 fe80::fca2:57ff:fe1c:67df  prefixlen 64  scopeid 0x20<link>
        ether fe:a2:57:1c:67:df  txqueuelen 1000  (以太网)
        RX packets 15  bytes 1146 (1.1 KB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 11  bytes 866 (866.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```


#### 验证`ns`内网络情况

从 `ns01`里`ping ns02`,同时在默认用`tcpdump`在`br01 bridge`上抓包

```bash
# 首先启动抓包
sudo tcpdump -i br01 -nn

tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
listening on br01, link-type EN10MB (Ethernet), capture size 262144 bytes

# 然后从 ns01 ping ns02
sudo ip netns exec ns01 ping 192.168.88.12 -c 1

PING 192.168.88.12 (192.168.88.12) 56(84) bytes of data.
64 bytes from 192.168.88.12: icmp_seq=1 ttl=64 time=0.086 ms

--- 192.168.88.12 ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 0.086/0.086/0.086/0.000 ms

# 查看抓包信息
16:19:42.739429 ARP, Request who-has 192.168.88.12 tell 192.168.88.11, length 28
16:19:42.739471 ARP, Reply 192.168.88.12 is-at fe:a2:57:1c:67:df, length 28
16:19:42.739476 IP 192.168.88.11 > 192.168.88.12: ICMP echo request, id 984, seq 1, length 64
16:19:42.739489 IP 192.168.88.12 > 192.168.88.11: ICMP echo reply, id 984, seq 1, length 64
16:19:47.794415 ARP, Request who-has 192.168.88.11 tell 192.168.88.12, length 28
16:19:47.794451 ARP, Reply 192.168.88.11 is-at d2:88:ec:62:cd:0a, length 28
```

可以看到`ARP`能正确定位到`MAC`地址,并且`reply`包能正确返回到`ns01`中,反之在`ns02`中`ping ns01`也是通的


在`ns01`内执行`arp`

```bash
sudo ip netns exec ns01 arp

地址                     类型    硬件地址            标志  Mask            接口
192.168.88.12            ether   fe:a2:57:1c:67:df   C                     veth01
192.168.88.1             ether   32:1b:c3:fd:56:fd   C                     veth01
```

可以看到`192.168.88.1`的`MAC`地址是正确的,跟`ip link`打印出来的是一致

```bash
ip link

6: br01: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default qlen 1000
    link/ether 32:1b:c3:fd:56:fd brd ff:ff:ff:ff:ff:ff
```

#### `ns`与外网互通

从`ns02 ping`  外网地址(如下以`114.114.114.114`为例子)

```bash
sudo ip netns exec ns02 ping 114.114.114.114 -c 1

PING 114.114.114.114 (114.114.114.114) 56(84) bytes of data.

--- 114.114.114.114 ping statistics ---
1 packets transmitted, 0 received, 100% packet loss, time 0ms
```

发现是`ping`不通的,抓包查看详情

```bash
# 抓Bridge设备
tcpdump -i br01 -nn -vv host 114.114.114.114

tcpdump: listening on br01, link-type EN10MB (Ethernet), capture size 262144 bytes
17:02:59.027478 IP (tos 0x0, ttl 64, id 51092, offset 0, flags [DF], proto ICMP (1), length 84)
    192.168.88.12 > 114.114.114.114: ICMP echo request, id 1045, seq 1, length 64


# 抓出口设备
tcpdump -i eth0 -nn -vv host 114.114.114.114
```

发现只有`br01`有出口流量,而出口网卡`eth0`没有任何反应,说明没有开启`ip_forward`

```bash
# 开启 ip_forward
sudo sysctl -w net.ipv4.conf.all.forwarding=1
```

再次尝试抓包`eth0`设备

```bash
sudo tcpdump -i eth0 -nn -vv host 114.114.114.114

tcpdump: listening on eth0, link-type EN10MB (Ethernet), capture size 262144 bytes
17:11:26.517292 IP (tos 0x0, ttl 63, id 15277, offset 0, flags [DF], proto ICMP (1), length 84)
    192.168.88.12 > 114.114.114.114: ICMP echo request, id 1059, seq 1, length 64
```

发现只有发出去的包`request`没有回来`replay`的包,原因是因为源地址是私有地址,如果发回来的包是私有地址会被丢弃
解决方法是将发出去的包`sourceIP`改成`gatewayIP`,可以用`SNAT`或者`MAQUERADE`

`SNAT`: 需要搭配静态IP
`MAQUERADE`: 可以用于动态分配的IP,但每次数据包被匹配中时,都会检查使用的IP地址

```bash
sudo iptables -t nat -A POSTROUTING -s 192.168.88.0/24 -j MASQUERADE

# 查看防火墙规则
sudo iptables -t nat -nL --line-number

Chain PREROUTING (policy ACCEPT)
num  target     prot opt source               destination         

Chain INPUT (policy ACCEPT)
num  target     prot opt source               destination         

Chain OUTPUT (policy ACCEPT)
num  target     prot opt source               destination         

Chain POSTROUTING (policy ACCEPT)
num  target     prot opt source               destination         
1    MASQUERADE  all  --  192.168.88.0/24      0.0.0.0/0
```

再次尝试`ping 114.114.114.114`

```bash
sudo ip netns exec ns02 ping 114.114.114.114 -c 1
```
抓包查看

```bash
sudo tcpdump -i eth0 -nn -vv host 114.114.114.114

tcpdump: listening on eth0, link-type EN10MB (Ethernet), capture size 262144 bytes
17:43:54.744599 IP (tos 0x0, ttl 63, id 46107, offset 0, flags [DF], proto ICMP (1), length 84)
    172.22.36.202 > 114.114.114.114: ICMP echo request, id 1068, seq 1, length 64
17:43:54.783749 IP (tos 0x4, ttl 71, id 62825, offset 0, flags [none], proto ICMP (1), length 84)
    114.114.114.114 > 172.22.36.202: ICMP echo reply, id 1068, seq 1, length 64

---

sudo tcpdump -i br01 -nn -vv
tcpdump: listening on br01, link-type EN10MB (Ethernet), capture size 262144 bytes17:43:54.744560 IP (tos 0x0, ttl 64, id 46107, offset 0, flags [DF], proto ICMP (1), length 84)
    192.168.88.12 > 114.114.114.114: ICMP echo request, id 1068, seq 1, length 64
17:43:54.783805 IP (tos 0x4, ttl 70, id 62825, offset 0, flags [none], proto ICMP (1), length 84)
    114.114.114.114 > 192.168.88.12: ICMP echo reply, id 1068, seq 1, length 64
```

可以看到从`eth0`出去的数据包的`sourceIP`已经变成网卡IP了
而`br01`收到的包的`sourceIP`还是`ns02` 的 `192.168.88.12`

#### 清理`iptables`规则

```bash
sudo iptables -t nat -F
```

#### 端对端映射

从上面可以得知,其实端口映射其实就是做一个`DNAT`
把宿主机的端口的数据包转发到内网的`ns`中

首先来创建一个`PREROUTING`规则的`chain`方便以后的转发规则的统筹

```bash
# 创建自定义链
sudo iptables -t nat -N TESTBR
```

创建回环, LOCAL 表示在主机的一个接口上可以分配任意IP,包括回环地址
比如本机的IP为`1.2.3.4`,但对于主机来说该`IP`指代主机本地,如果`ping 1.2.3.4`实际上是`lo`返回的响应包(可以亲自用`tcpdump`抓包验证)

```bash
sudo iptables -t nat -A PREROUTING -m addrtype --dst-type LOCAL -j TESTBR
sudo iptables -t nat -A OUTPUT ! -d 127.0.0.0/8 -m addrtype --dst-type LOCAL -j TESTBR
```


























---

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
