package datastore

import (
  "os"
  "io/ioutil"
  "encoding/json"
  log "github.com/Sirupsen/logrus"

)

const datastorePath = "/etc/docker/vde_plug_docker.data"

func New() {
  f, err := os.OpenFile(datastorePath, os.O_CREATE, 0644)
  if err != nil {
    log.Warnf("Datastore Create: [ %s ]", err)
    return
  }
  f.Close()
}

func Remove() {
  if err := os.Remove(datastorePath); err != nil {
    log.Warnf("Datastore Remove: [ %s ]", err)
  }
}

func Exists() bool {
  _, err := os.Stat(datastorePath)
  return (err == nil)
}

/* Check if the storage file exists and is not corrupt */
func Valid() bool {
  var buf interface{}
  data, err := ioutil.ReadFile(datastorePath)
  jsonerr := json.Unmarshal(data, &buf)
  return (err == nil && jsonerr == nil)
}

func Get(buf interface{}) {
  data, err := ioutil.ReadFile(datastorePath)
  if data != nil {
    err = json.Unmarshal(data, &buf)
  }
  if err != nil {
    log.Warnf("Datastore Get: [ %s ]", err)
  }
}

func Put(buf interface{}) {
  jsondata, err := json.Marshal(buf)
  if jsondata != nil {
    err = ioutil.WriteFile(datastorePath, jsondata, 0644)
  }
  if err != nil {
    log.Warnf("Datastore Put: [ %s ]", err)
  }
}
