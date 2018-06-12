# vde_plug_docker
VDE network plugin for Docker

### Dependencies
- go >= 1.7.4
- libvdeplug: https://github.com/rd235/vdeplug4

### Docker Hub install
```
$ docker plugin install phocs/vde
```

### Repo Install
```
$ cd vde_plug_docker/
$ make
$ sudo make install
```

### Debug
```
$ cd vde_plug_docker/
$ make
$ sudo ./vde_plug_docker --debug
```

### Example

```
$ docker network create -d vde -o sock=vxvde:// \
  --subnet 192.168.123.0/24 vdenet
$ docker run -it --net vdenet --ip 192.168.123.2 debian /bin/bash
```  
