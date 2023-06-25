# Tebis

This document focuses on setting up a development environment for the
distributed of Tebis on a local machine.


## Build Tebis Containers


Build the Tebis initialization manager:

```shell
docker build . -t tebis-init -f init.Dockerfile
```

Build the Tebis nodes:

```shell
docker build -t icsforth/tebis-node . -f tebis.Dockerfile
```


## Set up Execution Environment

Tebis uses RDMA for all network communication, which requires support from the
network interface to run. A software implementation (soft-RoCE) exists and can
run on all network interfaces.


### Install dependencies

The `ibverbs-utils` and `rdma-core` packages are required to enable soft-RoCE.
These packages should be in most distirbutions' repositories.

```
apt install ibverbs-utils rdma-core perftest
```

####

### Enabling soft-RoCE

Soft ROCE is a software implementation of RoCE that allows using Infiniband over any ethernet adapter.


ROCE requires the `ethtool` to be installed and the `rdma_ucm` and `uverbs0` modules to be loaded in your system.

```
sudo yum install ethtool
sudo modprobe rdma_rxe rdma_ucm
```

Then, where enp1s0 is the ethernet interface (e.g, eth0, en01, ...)

```
rdma link add rxe0 type rxe netdev enp1s0
```

validate it:

```
>> rdma link
link rxe0/1 state ACTIVE physical_state LINK_UP netdev eno1
```


#### Verify soft-RoCE is working
To verify that soft-RoCE is working, we can run a simple RDMA Write throuhgput
benchmark.

First, open two shells, one to act as the server and one to act as the client.
Then run the following commands:
* On the server: `ib_write_bw`
* On the client: `ib_write_bw eth_interface_ip`, where `eth_interface_ip` is
the IP address of a soft-RoCE enabled ethernet interface.

Example output:
* Server process:
```
************************************
* Waiting for client to connect... *
************************************
---------------------------------------------------------------------------------------
RDMA_Write BW Test
Dual-port       : OFF        Device         : rxe0
Number of qps   : 1        Transport type : IB
Connection type : RC        Using SRQ      : OFF
CQ Moderation   : 100
Mtu             : 1024[B]
Link type       : Ethernet
GID index       : 1
Max inline data : 0[B]
rdma_cm QPs     : OFF
Data ex. method : Ethernet
---------------------------------------------------------------------------------------
local address: LID 0000 QPN 0x0011 PSN 0x3341fd RKey 0x000204 VAddr 0x007f7e1b8fa000
GID: 00:00:00:00:00:00:00:00:00:00:255:255:192:168:122:205
remote address: LID 0000 QPN 0x0012 PSN 0xbfbac5 RKey 0x000308 VAddr 0x007f70f5843000
GID: 00:00:00:00:00:00:00:00:00:00:255:255:192:168:122:205
---------------------------------------------------------------------------------------
#bytes     #iterations    BW peak[MB/sec]    BW average[MB/sec]   MsgRate[Mpps]
65536      5000             847.44             827.84           0.013245
---------------------------------------------------------------------------------------
```

* Client process:
```
---------------------------------------------------------------------------------------
RDMA_Write BW Test
Dual-port       : OFF        Device         : rxe0
Number of qps   : 1        Transport type : IB
Connection type : RC        Using SRQ      : OFF
TX depth        : 128
CQ Moderation   : 100
Mtu             : 1024[B]
Link type       : Ethernet
GID index       : 1
Max inline data : 0[B]
rdma_cm QPs     : OFF
Data ex. method : Ethernet
---------------------------------------------------------------------------------------
local address: LID 0000 QPN 0x0012 PSN 0xbfbac5 RKey 0x000308 VAddr 0x007f70f5843000
GID: 00:00:00:00:00:00:00:00:00:00:255:255:192:168:122:205
remote address: LID 0000 QPN 0x0011 PSN 0x3341fd RKey 0x000204 VAddr 0x007f7e1b8fa000
GID: 00:00:00:00:00:00:00:00:00:00:255:255:192:168:122:205
---------------------------------------------------------------------------------------
#bytes     #iterations    BW peak[MB/sec]    BW average[MB/sec]   MsgRate[Mpps]
65536      5000             847.44             827.84           0.013245
---------------------------------------------------------------------------------------
```


```shell
server: ibv_rc_pingpong -d rxe0 -g 1
client: ibv_rc_pingpong -d rxe0 -g 1 10.1.128.51
```


### Others


Run rxe_cfg add ethN to configure an RXE instance on ethernet device ethN.

``` shell
You should now have an rxe0 device:

# rxe_cfg status

Name    Link  Driver      Speed  NMTU  IPv4_addr        RDEV  RMTU
enp1s0  yes   virtio_net         1500  192.168.122.211  rxe0  1024  (3)
```


https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html-single/configuring_infiniband_and_rdma_networks/index
https://support.mellanox.com/s/article/howto-configure-soft-roce
https://github.com/ememos/GiantVM/issues/24


On /etc/security/limits.conf you must add

```
*            soft    memlock         unlimited
*            hard    memlock         unlimited
```

https://bbs.archlinux.org/viewtopic.php?id=273059

https://ask.cyberinfrastructure.org/t/access-to-dev-infiniband-from-user-space/854/2


## Parameters
