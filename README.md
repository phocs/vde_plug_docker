# vde_plug_docker

vde_plug_docker implements the concept of Virtual Distributed Container. It's a Docker Network Plug-in that allows communication between containers on the VDE network.

The plugin is responsible for providing the access point to the VDE network. The construction and management of the latter it's outside the plugin's competence. 

### Doc References

https://drive.google.com/drive/folders/1nV5Kku1676_cLKRgJ8m9d7ozMa9X7VVi

### Dependencies
- go >= 1.7.4
- libvdeplug: https://github.com/rd235/vdeplug4

### Docker Hub install

You can install the pre-built image of the plugin from the Docker Hub. Is required:

- Host network access. 
- Host  /tmp mounted.
- Device access /dev/net/tun.
- CAP_NET_ADMIN capabilities
```
$ docker plugin install phocs/vde
```

### Git Install

In this way the plugin is installed as a system-daemon according to the [Docker Docs].

[Docker Docs]: https://docs.docker.com/v17.09/engine/extend/plugin_api/#json-specification
```
$ cd vde_plug_docker/
$ make
$ sudo make install
```

### Debug

You can also manually run the plugin for debugging and development.
```
$ cd vde_plug_docker/
$ make
$ sudo ./vde_plug_docker --debug
```

### Examples

#### Connect 2 containers to the VXVDE network

Create Network:
```
# docker network create -d vde \
  -o sock=vxvde://239.1.2.3 \
  --subnet 10.10.0.1/24 vdenet
```

Run containers:
```
# docker run -it --net vdenet --ip 10.10.0.2 debian
# docker run -it --net vdenet --ip 10.10.0.3 debian
# ping 10.10.0.2
   PING 10.10.0.2 (10.10.0.2) 56(84) bytes of data.
    64 bytes from 10.10.0.2: icmp_seq=1 ttl=64 time=0.573 ms
    64 bytes from 10.10.0.2: icmp_seq=2 ttl=64 time=0.373 ms
    64 bytes from 10.10.0.2: icmp_seq=3 ttl=64 time=0.293 ms
    64 bytes from 10.10.0.2: icmp_seq=4 ttl=64 time=0.365 ms

```

#### Add a VM to the network

```
$ kvm ... -net nic,macaddr=52:54:00:11:22:11 -net vde,sock=vxvde://239.1.2.3
# ip link set eth0 up
# ip addr add 10.10.0.42/24 dev eth0
# ping 10.10.0.2          [container 1]
   PING 10.10.0.2 (10.10.0.2) 56(84) bytes of data.
    64 bytes from 10.10.0.2: icmp_seq=1 ttl=64 time=0.674 ms
    64 bytes from 10.10.0.2: icmp_seq=2 ttl=64 time=0.252 ms
    64 bytes from 10.10.0.2: icmp_seq=3 ttl=64 time=0.332 ms
    64 bytes from 10.10.0.2: icmp_seq=4 ttl=64 time=0.472 ms
```
