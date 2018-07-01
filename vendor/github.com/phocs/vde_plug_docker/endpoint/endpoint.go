package endpoint

//#cgo LDFLAGS: -lvdeplug -lpthread
//#include <vdeplug.h>
import "C"
import (
  "net"
  "errors"
  "crypto/rand"
  log "github.com/Sirupsen/logrus"
  "github.com/vishvananda/netlink"
  "github.com/docker/go-plugins-helpers/network"
)

type EndpointStat struct {
  Plugger         uintptr `json:"Plugger"`
  IfName          string  `json:"IfName"`
  SandboxKey      string  `json:"SandboxKey"`
  IPv4Address     string  `json:"IPv4Address"`
	IPv6Address     string  `json:"IPv6Address"`
  MacAddress      string  `json:"MacAddress"`
}

func NewEndpointStat(r *network.CreateEndpointRequest) (*EndpointStat) {
  new := EndpointStat{
    Plugger:      0,
    IfName:       "vde" + r.EndpointID[:11],
    SandboxKey:   "",
    IPv4Address:  r.Interface.Address,
    IPv6Address:  r.Interface.AddressIPv6,
    MacAddress:   r.Interface.MacAddress,
  }
  if new.MacAddress == "" {
    new.MacAddress = RandomMacAddr()
  }
  return &new
}

func (this *EndpointStat) LinkAdd() error {
  linkattrs := netlink.NewLinkAttrs()
  linkattrs.Name = this.IfName
  linkattrs.HardwareAddr, _ = net.ParseMAC(this.MacAddress)

  tapdev := &netlink.Tuntap{LinkAttrs: linkattrs}
  tapdev.Flags = netlink.TUNTAP_NO_PI
  tapdev.Mode =  netlink.TUNTAP_MODE_TAP

  if err := netlink.LinkAdd(tapdev); err != nil {
    return err
  }
  if ipv4, err := netlink.ParseAddr(this.IPv4Address); err == nil {
    netlink.AddrAdd(tapdev, ipv4)
  }
  if ipv6, err := netlink.ParseAddr(this.IPv6Address); err == nil {
    netlink.AddrAdd(tapdev, ipv6)
  }
  return nil
}

func (this *EndpointStat) LinkDel() error {
  var err error
  if link, err := netlink.LinkByName(this.IfName); err == nil {
    err = netlink.LinkDel(link)
  }
  return err
}

func (this *EndpointStat) LinkPlugTo(sock string) error {
  log.Debugf("LinkPlugTo [ %s ] [ %s ]", this.IfName, sock)
  this.Plugger = uintptr(C.vdeplug_join(C.CString(this.IfName), C.CString(sock)))
  if this.Plugger == 0 {
    return errors.New("LinkPlugTo error: " + this.IfName + " to " + sock)
  }
  return nil
}

func (this *EndpointStat) LinkPlugStop() {
  C.vdeplug_leave(C.uintptr_t(this.Plugger));
  this.Plugger = 0
}

/*Copied from include/linux/etherdevice.h
  This is the kernel's method of making random mac addresses */
func RandomMacAddr() string {
	mac := make([]byte, 6)
	rand.Read(mac)
	mac[0] &= 0xfe; /* clear multicast bit */
	mac[0] |= 0x02  /* set local assignment bit (IEEE802) */
	return net.HardwareAddr(mac).String()
}
