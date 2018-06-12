package endpoint

//#cgo LDFLAGS: -lvdeplug -lpthread
//#include <vdeplug.h>
import "C"
import (
  "net"
  "errors"
  "crypto/rand"
  "github.com/vishvananda/netlink"
  "github.com/docker/go-plugins-helpers/network"
  log "github.com/Sirupsen/logrus"
)

type EndpointStat struct {
  Vdeplug         uintptr `json:"-"` // ignore
  IfName          string  `json:"IfName"`
  SandboxKey      string  `json:"SandboxKey"`
  IPv4Address     string  `json:"IPv4Address"`
	IPv6Address     string  `json:"IPv6Address"`
  MacAddress      string  `json:"MacAddress"`
}

func NewEndpointStat(r *network.CreateEndpointRequest) (*EndpointStat) {
  new := EndpointStat{
    Vdeplug:      0,
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

func (this *EndpointStat) AddLink() error {
  linkattrs := netlink.NewLinkAttrs()
  linkattrs.Name = this.IfName
  linkattrs.HardwareAddr, _ = net.ParseMAC(this.MacAddress)

  tapdev := &netlink.Tuntap{LinkAttrs: linkattrs}
  tapdev.Flags = netlink.TUNTAP_NO_PI
  tapdev.Mode =  netlink.TUNTAP_MODE_TAP

  if ipv4, err := netlink.ParseAddr(this.IPv4Address); err == nil {
    netlink.AddrAdd(tapdev, ipv4)
  }
  if ipv6, err := netlink.ParseAddr(this.IPv6Address); err == nil {
    netlink.AddrAdd(tapdev, ipv6)
  }
  return netlink.LinkAdd(tapdev)
}

func (this *EndpointStat) DelLink() {
  link, _ := netlink.LinkByName(this.IfName)
  netlink.LinkDel(link)
}

func (this *EndpointStat) PlugLinkTo(sock string) error {
  log.Debugf("PlugLinkTo [ %s ] [ %s ]",   this.IfName, sock)
  this.Vdeplug = uintptr(C.vdeplug_start(C.CString(this.IfName), C.CString(sock)))
  if this.Vdeplug == 0 {
    return errors.New("PlugLinkTo error: " + this.IfName + " to " + sock)
  }
  return nil
}

func (this *EndpointStat) PlugStop() {
  C.vdeplug_stop(C.uintptr_t(this.Vdeplug));
  this.Vdeplug = 0
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
