Sometimes Tebis crashes while trying to calculate tail latency. To disable, change the git cloning within the Dockerfile
to the following command:


Build the Tebis initialization manager
```
docker build . -t tebis-init -f init.Dockerfile
```

Build the Tebis nodes
```
docker build -t icsforth/tebis-node . -f tebis.Dockerfile
```

# Install a Virtual RDMA NIC using Soft RoCE (RXE)
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






Run rxe_cfg start to load RXE modules and configure persistent instances.
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


server: ibv_rc_pingpong -d rxe0 -g 1
client: ibv_rc_pingpong -d rxe0 -g 1 10.1.128.51