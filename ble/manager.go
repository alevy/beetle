package ble

import (
  "errors"
  "time"
  "io"
  "os"
)

type ManagerRequest struct {
  msg []byte
  device *Device
}

type Manager struct {
  Devices []*Device
  globalHandleOffset int
  associations map[int][]int
  requestChan chan ManagerRequest
  hciSock     *os.File
  log         *os.File
}

func NewManager(hciSock *os.File) (*Manager) {
  f, _ := os.Create("log.csv")
  return &Manager{make([]*Device, 0), 0, make(map[int][]int),
    make(chan ManagerRequest), hciSock, f}
}

func (this *Manager) ConnectTo(addr string) error {
  remoteAddr, err := Str2Ba(addr)
  if err != nil {
    return err
  }

  f, err := NewBLE(NewL2Sockaddr(4, remoteAddr, BDADDR_LE_PUBLIC), addr)
  if err != nil {
    return err
  }

  ci := GetConnInfo(f)

  this.AddDeviceForConn(addr, f, ci)

  return nil
}

func (this *Manager) ConnUpdate(device *Device, interval uint16) int {
  if device.connInfo != nil {
    return HCIConnUpdate(this.hciSock, device.connInfo.HCIHandle, interval, interval, 0, 0x0C80)
  } else {
    return 0
  }
}

func (this *Manager) AddDeviceForConn(addr string, f io.ReadWriteCloser, ci *ConnInfo) (*Device) {
  device := NewDevice(addr, this.requestChan, f, ci)
  this.Devices = append(this.Devices, device)
  return device
}

func (this *Manager) StartNoDiscover(idx int) error {
  if idx >= len(this.Devices) || idx < 0 {
    return errors.New("No such device")
  }

  device := this.Devices[idx]
  return this.StartDeviceNoDiscover(device)
}

func (this *Manager) StartDeviceNoDiscover(device *Device) error {
  device.Start()
  return nil
}

func (this *Manager) Start(idx int) error {
  if idx >= len(this.Devices) || idx < 0 {
    return errors.New("No such device")
  }
  device := this.Devices[idx]
  return this.StartDevice(device)
}

func (this *Manager) StartDevice(device *Device) error {
  device.Start()

  handles, err := DiscoverHandles(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  device.handleOffset = this.globalHandleOffset
  this.globalHandleOffset += len(handles)

  for _, handle := range handles {
    h := &Handle{*handle, time.Now(), nil, nil}
    device.handles[handle.handle] = h
  }

  groupVals, err := DiscoverServices(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  for _,v := range groupVals {
    device.handles[v.handle].cachedValue = v.value
    device.handles[v.handle].endGroup = v.endGroup
  }

  handleVals, err := DiscoverCharacteristics(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  for _,v := range handleVals {
    device.handles[v.handle].cachedValue = v.value
  }

  return nil
}

func (this *Manager) DisconnectFrom(idx int) error {
  if idx >= len(this.Devices) || idx < 0 {
    return errors.New("No such device")
  }

  device := this.Devices[idx]

  err := device.fd.Close()
  if err != nil {
    return err
  }

  this.Devices = append(this.Devices[0:idx], this.Devices[idx + 1:]...)
  delete(this.associations, idx)
  for cl,lst := range this.associations {
    for i,v := range lst {
      if v == idx {
        this.associations[cl] = append(lst[0:i], lst[i + 1:]...)
        break
      }
    }
    if len(this.associations) == 0 {
      delete(this.associations, cl)
    }
  }
  return nil
}

func (this *Manager) ServeTo(serverIdx, clientIdx int) error {
  if clientIdx >= len(this.Devices) || serverIdx >= len(this.Devices) {
    return errors.New("No such device")
  }

  if lst,ok := this.associations[clientIdx]; ok {
    this.associations[clientIdx] = append(lst, serverIdx)
  } else {
    this.associations[clientIdx] = []int {serverIdx}
  }

  return nil
}

