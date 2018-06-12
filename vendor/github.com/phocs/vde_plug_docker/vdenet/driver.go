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
  mutex     sync.Mutex                `json:"-"` // ignore
  Networks  map[string]*NetworkStat   `json:"Networks"`
}

const (
  IfPrefixDefault = "vde"
  RequestOpts = "com.docker.network.generic"
)

func NewDriver() Driver {
  driver := Driver {
    Networks: make(map[string]*NetworkStat),
  }
  if datastore.Exists() && datastore.Valid() {
    datastore.Get(&driver)
    DriverCheck(&driver)
    return driver
  } else if datastore.Exists() {
    datastore.Remove()
    log.Debugf("Removed corrupt Datastore")
  }
  datastore.New()
  return driver
}

/* Delete "active" endpoints */
func DriverCheck(d *Driver) {
  for nwkey, nw := range d.Networks {
    for edpkey, edp := range nw.Endpoints {
      if edp.Vdeplug == 0 {
        delete(d.Networks[nwkey].Endpoints, edpkey)
        datastore.Put(d)
      }
    }
  }
}

func (this *Driver) UpdateNUnlock() {
  go func() {
    datastore.Put(this)
    this.mutex.Unlock()
  }()
}

/* CapabilitiesResponse returns whether or not this network is global or local, */
func (this *Driver) GetCapabilities() (*network.CapabilitiesResponse, error) {
  return &network.CapabilitiesResponse{Scope: network.LocalScope}, nil
}

func (this *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
  log.Debugf("CREATE NETWORK: [ %+v ]", r)

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
  log.Debugf("Sock [ %s ]", sock)

  this.mutex.Lock()
  defer this.mutex.Lock()
  this.Networks[r.NetworkID] = &NetworkStat{
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
  log.Debugf("Allocatenetwork Request: [ %+v ]", r)
  return nil, types.NotImplementedErrorf("Not implementethis.")
}

func (this *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
  log.Debugf("Deletenetwork: [ %+v ]", r)

  var netw *NetworkStat
	if netw = this.Networks[r.NetworkID]; netw == nil {
    return types.NotFoundErrorf("Network not founthis.")
	}
  if len(netw.Endpoints) == 0 {
    this.mutex.Lock()
defer this.UpdateNUnlock()
    delete(this.Networks, r.NetworkID)
    return nil
  }
  return types.BadRequestErrorf("There are still active endpoints.")
}

func (this *Driver) FreeNetwork(r *network.FreeNetworkRequest) error {
  log.Warnf("Freenetwork Request: [ %+v ]", r)
	return types.NotImplementedErrorf("Not implementethis.")
}

func (this *Driver) CreateEndpoint(r *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
  log.Debugf("CREATE ENDPOINT: [ %+v ]", r)

  netw := this.Networks[r.NetworkID]
  if netw == nil {
    return nil, types.NotFoundErrorf("Network not found.")
  }
  if netw.Endpoints[r.EndpointID] != nil {
    return nil, types.BadRequestErrorf("EndpointID already exists.")
  }
  this.mutex.Lock()
defer this.UpdateNUnlock()
  netw.Endpoints[r.EndpointID] = endpoint.NewEndpointStat(r)
  response := &network.CreateEndpointResponse{
    Interface: &network.EndpointInterface{},
  }
  if r.Interface.MacAddress == "" {
     response.Interface.MacAddress = netw.Endpoints[r.EndpointID].MacAddress
  }
  return response, nil
}

func (this *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {
  log.Debugf("DeleteEndpoint: [ %+v ]", r)

  if this.Networks[r.NetworkID] == nil {
    return types.NotFoundErrorf("Network not found.")
  }
  if this.Networks[r.NetworkID].Endpoints[r.EndpointID] == nil {
    return types.NotFoundErrorf("Endpoint not found.")
  }
  this.mutex.Lock()
defer this.UpdateNUnlock()
  this.Networks[r.NetworkID].Endpoints[r.EndpointID].DelLink()
  delete(this.Networks[r.NetworkID].Endpoints, r.EndpointID)
  return nil
}

func (this *Driver) EndpointInfo(r *network.InfoRequest) (*network.InfoResponse, error) {
  log.Debugf("ENDPOINT INFO: [ %+v ]", r)

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
  if netw = this.Networks[r.NetworkID]; netw == nil {
    return nil, types.NotFoundErrorf("network not found.")
  }
  if edpt = netw.Endpoints[r.EndpointID]; edpt == nil {
    return nil, types.NotFoundErrorf("Endpoint not found.")
  }

  this.mutex.Lock()
defer this.UpdateNUnlock()
  if edpt.AddLink() != nil {
    return nil, types.RetryErrorf("Failed link create.")
  }
  if err := edpt.PlugLinkTo(netw.Sock); err != nil {
    return nil, types.NotFoundErrorf("Failed plug to interface", err)
  }
  if netw.IPv4Gateway != "" {
    gateway = net.ParseIP(strings.Split(netw.IPv4Gateway, "/")[0]).String()
  }
  if netw.IPv6Gateway != "" {
    gateway6 = net.ParseIP(strings.Split(netw.IPv6Gateway, "/")[0]).String()
  }
  return &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   edpt.IfName,
			DstPrefix: netw.IfPrefix,
		},
		Gateway:     gateway,
		GatewayIPv6: gateway6,
  }, nil
}

func (this *Driver) Leave(r *network.LeaveRequest) error {
  log.Debugf("LEAVE: [ %+v ]", r)

  var netw *NetworkStat
  var edpt *endpoint.EndpointStat
  if netw = this.Networks[r.NetworkID]; netw == nil {
    return types.NotFoundErrorf("network not found.")
  }
  if edpt = netw.Endpoints[r.EndpointID]; edpt == nil {
    return types.NotFoundErrorf("Endpoint not found.")
  }
  this.mutex.Lock()
defer this.UpdateNUnlock()
  edpt.PlugStop()
  return nil
}

func (this *Driver) DiscoverNew(r *network.DiscoveryNotification) error {
  log.Debugf("DISCOVER NEW Called: [ %+v ]", r)
  return nil
}

func (this *Driver) DiscoverDelete(r *network.DiscoveryNotification) error {
  log.Debugf("DISCOVER DELETE Called: [ %+v ]", r)
  return nil
}

func (this *Driver) ProgramExternalConnectivity(r *network.ProgramExternalConnectivityRequest) error {
  log.Debugf("PROGRAM EXTERNAL CONNECTIVITY Called: [ %+v ]", r)
  return nil
}

func (this *Driver) RevokeExternalConnectivity(r *network.RevokeExternalConnectivityRequest) error {
  log.Debugf("REVOKE EXTERNAL CONNECTIVITY Called: [ %+v ]", r)
  return nil
}
