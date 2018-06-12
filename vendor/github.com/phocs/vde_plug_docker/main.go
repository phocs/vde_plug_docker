
package main

import (
  log "github.com/Sirupsen/logrus"
  "gopkg.in/alecthomas/kingpin.v2"
  "github.com/phocs/vde_plug_docker/vdenet"
  "github.com/docker/go-plugins-helpers/network"
)

const unixSock = "/run/docker/plugins/vde.sock"

func main() {
  debug := kingpin.Flag("debug", "Enable debug mode.").Bool()
	kingpin.Parse()

  if *debug {
    log.SetLevel(log.DebugLevel)
  }

  d := vdenet.NewDriver()
  h := network.NewHandler(&d)
  if err := h.ServeUnix(unixSock, 0); err != nil {
    log.Fatal(err)
  }
}
