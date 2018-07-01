package vdenet

import (
  "net"
  "sync"
  "strings"
  log "github.com/Sirupsen/logrus"
  "github.com/docker/libnetwork/types"
  "github.com/phocs/vde_plug_docker/endpoint"
  "github.com/phocs/vde_plug_docker/datastore"
  "github.com/docker/go-plugins-helpers/network"
)

type NetworkStat struct {
  Sock          string                            `json:"Sock"`
  IfPrefix      string                            `json:"IfPrefix"`
  IPv4Pool      string                            `json:"IPv4Pool"`
  IPv4Gateway   string                            `json:"IPv4Gateway"`
  IPv6Pool      string                            `json:"IPv6Pool"`
  IPv6Gateway   string                            `json:"IPv6Gateway"`
  Endpoints     map[string]*endpoint.EndpointStat `json:"Endpoints"`
}

type Driver struct {
  mutex     sync.RWMutex              `json:"-"` // ignore
  Networks  map[string]*NetworkStat   `json:"Networks"`
}

const (
  IfPrefixDefault = "vde"
)

func NewDriver(storepath string, clean bool) Driver {
  driver := Driver { Networks: make(map[string]*NetworkStat), }
  datastore.SetPath(storepath)
  if clean == true {
    datastore.Clean()
  } else if err := datastore.Load(&driver); err == nil {
    /* Check the old Driver data */
    for nwkey, nw := range driver.Networks {
      for epkey, ep := range nw.Endpoints {
        if ep.Plugger == 0 || ep.LinkDel() == nil {
          /* Container has been stopped or is running (whitout plugger) */
          delete(driver.Networks[nwkey].Endpoints, epkey)
        }
      }
    }
    _ = datastore.Store(&driver)
  }
  return driver
}

/* CapabilitiesResponse returns whether or not this network is global or local, */
func (this *Driver) GetCapabilities() (*network.CapabilitiesResponse, error) {
  return &network.CapabilitiesResponse{ Scope: network.GlobalScope }, nil
}

func (this *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
  var sock, ifprefix, ipv6pool, ipv6gateway string
  opt := r.Options["com.docker.network.generic"].(map[string]interface{})
  if r.IPv4Data == nil || len(r.IPv4Data) == 0 {
		return types.BadRequestErrorf("Network IPv4Data config miss.")
	}
  if sock, _ = opt["sock"].(string); sock == "" {
    return types.NotFoundErrorf("Sock URL miss.")
  }
  if ifprefix, _ = opt["if"].(string); ifprefix == "" {
    ifprefix = IfPrefixDefault
  }
  if r.IPv6Data != nil && len(r.IPv6Data) > 0 {
    ipv6pool = r.IPv6Data[0].Pool
    ipv6gateway = r.IPv6Data[0].Gateway
  }
  this.mutex.Lock()
  defer this.mutex.Unlock()
  defer datastore.Store(&this)
  this.Networks[r.NetworkID] = &NetworkStat {
    Sock:         sock,
    IfPrefix:     ifprefix,
    IPv4Pool:     r.IPv4Data[0].Pool,
    IPv4Gateway:  r.IPv4Data[0].Gateway,
    IPv6Pool:     ipv6pool,
    IPv6Gateway:  ipv6gateway,
    Endpoints:    make(map[string]*endpoint.EndpointStat),
  }
  return nil
}

func (this *Driver) AllocateNetwork(r *network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
//  log.Debugf("Allocatenetwork Request: [ %+v ]", r)
  return nil, types.NotImplementedErrorf("Not implementethis.")
}

func (this *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
  log.Debugf("Deletenetwork: [ %+v ]", r)
  var netw *NetworkStat
  this.mutex.Lock()
  defer this.mutex.Unlock()
	if netw = this.Networks[r.NetworkID]; netw == nil {
    return types.NotFoundErrorf("Network not found.")
	}
  if len(netw.Endpoints) != 0 {
    return types.BadRequestErrorf("There are still active endpoints.")
  }
  delete(this.Networks, r.NetworkID)
  _ = datastore.Store(&this)
  return nil
}

func (this *Driver) FreeNetwork(r *network.FreeNetworkRequest) error {
//  log.Warnf("Freenetwork Request: [ %+v ]", r)
	return types.NotImplementedErrorf("Not implementethis.")
}

func (this *Driver) CreateEndpoint(r *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
  log.Debugf("CREATE ENDPOINT: [ %+v ]", r)
  this.mutex.Lock()
  defer this.mutex.Unlock()
  netw := this.Networks[r.NetworkID]
  if netw == nil {
    return nil, types.NotFoundErrorf("Network not found.")
  }
  if netw.Endpoints[r.EndpointID] != nil {
    return nil, types.BadRequestErrorf("EndpointID already exists.")
  }
  netw.Endpoints[r.EndpointID] = endpoint.NewEndpointStat(r)
  response := &network.CreateEndpointResponse {
    Interface: &network.EndpointInterface{},
  }
  if r.Interface.MacAddress == "" {
     response.Interface.MacAddress = netw.Endpoints[r.EndpointID].MacAddress
  }
  _ = datastore.Store(&this)
  return response, nil
}

func (this *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {
  log.Debugf("DeleteEndpoint: [ %+v ]", r)
  this.mutex.Lock()
  defer this.mutex.Unlock()
  if this.Networks[r.NetworkID] == nil {
    return types.NotFoundErrorf("Network not found.")
  }
  if this.Networks[r.NetworkID].Endpoints[r.EndpointID] == nil {
    return types.NotFoundErrorf("Endpoint not found.")
  }
  this.Networks[r.NetworkID].Endpoints[r.EndpointID].LinkDel()
  delete(this.Networks[r.NetworkID].Endpoints, r.EndpointID)
  _ = datastore.Store(&this)
  return nil
}

func (this *Driver) EndpointInfo(r *network.InfoRequest) (*network.InfoResponse, error) {
  log.Debugf("ENDPOINT INFO: [ %+v ]", r)
  this.mutex.RLock()
  defer this.mutex.RUnlock()
  if this.Networks[r.NetworkID] == nil {
    return nil, types.NotFoundErrorf("network not found.")
  }
  if this.Networks[r.NetworkID].Endpoints[r.EndpointID] == nil {
    return nil, types.NotFoundErrorf("Endpoint not found.")
  }
  info := &network.InfoResponse{ Value: make(map[string]string) }
  info.Value["id"]      = r.EndpointID
  info.Value["srcName"] = this.Networks[r.NetworkID].Endpoints[r.EndpointID].IfName
  return info, nil
}

func (this *Driver) Join(r *network.JoinRequest) (*network.JoinResponse, error) {
  log.Debugf("JOIN: [ %+v ]", r)
  var netw *NetworkStat
  var edpt *endpoint.EndpointStat;
  var gateway, gateway6 string

  this.mutex.Lock()
  defer this.mutex.Unlock()
  if netw = this.Networks[r.NetworkID]; netw == nil {
    return nil, types.NotFoundErrorf("Network not found.")
  }
  if edpt = netw.Endpoints[r.EndpointID]; edpt == nil {
    return nil, types.NotFoundErrorf("Endpoint not found.")
  }
  if edpt.LinkAdd() != nil {
    return nil, types.RetryErrorf("Failed link create.")
  }
  if err := edpt.LinkPlugTo(netw.Sock); err != nil {
    edpt.LinkDel()
    return nil, types.NotFoundErrorf("Failed plug to interface", err)
  }
  if netw.IPv4Gateway != "" {
    gateway = net.ParseIP(strings.Split(netw.IPv4Gateway, "/")[0]).String()
  }
  if netw.IPv6Gateway != "" {
    gateway6 = net.ParseIP(strings.Split(netw.IPv6Gateway, "/")[0]).String()
  }
  response := &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   edpt.IfName,
			DstPrefix: netw.IfPrefix,
		},
		Gateway:     gateway,
		GatewayIPv6: gateway6,
  }
  _ = datastore.Store(&this)
  return response, nil
}

func (this *Driver) Leave(r *network.LeaveRequest) error {
  log.Debugf("LEAVE: [ %+v ]", r)
  var netw *NetworkStat
  var edpt *endpoint.EndpointStat

  this.mutex.Lock()
  defer this.mutex.Unlock()
  if netw = this.Networks[r.NetworkID]; netw == nil {
    return types.NotFoundErrorf("network not found.")
  }
  if edpt = netw.Endpoints[r.EndpointID]; edpt == nil {
    return types.NotFoundErrorf("Endpoint not found.")
  }
  edpt.LinkPlugStop()
  edpt.LinkDel()
  _ = datastore.Store(&this)
  return nil
}

func (this *Driver) DiscoverNew(r *network.DiscoveryNotification) error {
//  log.Debugf("DISCOVER NEW Called: [ %+v ]", r)
  return nil
}

func (this *Driver) DiscoverDelete(r *network.DiscoveryNotification) error {
//  log.Debugf("DISCOVER DELETE Called: [ %+v ]", r)
  return nil
}

func (this *Driver) ProgramExternalConnectivity(r *network.ProgramExternalConnectivityRequest) error {
//  log.Debugf("PROGRAM EXTERNAL CONNECTIVITY Called: [ %+v ]", r)
  return nil
}

func (this *Driver) RevokeExternalConnectivity(r *network.RevokeExternalConnectivityRequest) error {
//  log.Debugf("REVOKE EXTERNAL CONNECTIVITY Called: [ %+v ]", r)
  return nil
}
