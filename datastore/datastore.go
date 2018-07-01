package datastore

import (
  "sync"
  "io/ioutil"
  "encoding/json"
  log "github.com/Sirupsen/logrus"
)

type DataStore struct {
  sync.Mutex
  Path  string
}

const OpenMode = 0644

var store = DataStore {
  Path: "./datastore.json",
}

func SetPath(path string) {
  store.Path = path
}

func Clean() {
  store.Lock()
  defer store.Unlock()
  if err := ioutil.WriteFile(store.Path, nil, OpenMode); err != nil {
    log.Warnf("Datastore.Clean: [ %s ]", err)
  }
}

func Load(elem interface{}) error {
  var err error
  store.Lock()
  defer store.Unlock()
  if buf, err := ioutil.ReadFile(store.Path); err == nil {
    return json.Unmarshal(buf, &elem)
  } else {
    log.Warnf("Datastore.Load: [ %s ]", err)
  }
  return err
}

func Store(elem interface{}) error {
  var err error
  if buf, err := json.Marshal(&elem); err == nil {
    store.Lock()
    err = ioutil.WriteFile(store.Path, buf, OpenMode)
    store.Unlock()
  }
  if err != nil {
    log.Warnf("Datastore.Store: [ %s ]", err)
  }
  return err
}
