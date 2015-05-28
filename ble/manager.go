package ble

import (
  "errors"
  "time"
  "io"
  "net"
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
}

func NewManager(hciSock *os.File) (*Manager) {
  return &Manager{make([]*Device, 0), 0, make(map[int][]int),
    make(chan ManagerRequest), hciSock}
}

func (this *Manager) ConnectTo(addrType uint8, addr string) error {
  remoteAddr, err := Str2Ba(addr)
  if err != nil {
    return err
  }

  f, err := NewBLE(NewL2Sockaddr(4, remoteAddr, addrType), addr)
  if err != nil {
    return err
  }

  ci := GetConnInfo(f)

  this.AddDeviceForConn(addr, f, ci)

  return nil
}

func (this *Manager) ConnectTCP(addr string) (error) {
  conn, err := net.Dial("tcp", addr)
  if err != nil {
    return err
  }
  this.AddDeviceForConn("tcp://" + addr, conn, nil)
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

  lastHandle := uint16(0)

  services, err := DiscoverServices(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  for _,service := range services {
    handle := new(Handle)
    handle.handle = service.handle
    handle.uuid = GATT_PRIMARY_SERVICE_UUID
    handle.cachedTime = time.Now()
    handle.cachedInfinite = true
    handle.cachedValue = service.value
    handle.endGroup = service.endGroup
    device.handles[service.handle] = handle
    chars, err := DiscoverCharacteristics(device, service.handle,
                       service.endGroup)
    if err != nil {
      device.fd.Close()
      return err
    }
    for _,char := range chars {
      handle := new(Handle)
      handle.handle = char.handle
      handle.uuid = GATT_CHARACTERISTIC_UUID
      handle.cachedTime = time.Now()
      handle.cachedInfinite = true
      handle.cachedValue = char.value
      handle.serviceHandle = service.handle
      handle.charHandle = uint16(char.value[1]) + (uint16(char.value[2]) << 8)
      device.handles[char.handle] = handle
    }

    for i := 0; i < len(chars) - 1; i++ {
      char := chars[i]
      startGroup := char.handle + 1
      endGroup := chars[i + 1].handle
      device.handles[char.handle].endGroup = endGroup
      handleInfos, err := DiscoverHandles(device, startGroup, endGroup)
      if err != nil {
        device.fd.Close()
        return err
      }
      for _, handleInfo := range(handleInfos) {
        handle := new(Handle)
        handle.handle = handleInfo.handle
        handle.uuid = handleInfo.uuid
        handle.cachedInfinite = false
        handle.serviceHandle = service.handle
        handle.charHandle = char.handle
        device.handles[handleInfo.handle] = handle
      }
    }

    if len(chars) == 0 {
      continue
    }

    char := chars[len(chars) - 1]
    startGroup := char.handle + 1
    endGroup := service.endGroup
    device.handles[char.handle].endGroup = endGroup
    handleInfos, err := DiscoverHandles(device, startGroup, endGroup)
    if err != nil {
      device.fd.Close()
      return err
    }
    for _, handleInfo := range(handleInfos) {
      handle := new(Handle)
      handle.handle = handleInfo.handle
      handle.uuid = handleInfo.uuid
      handle.cachedInfinite = false
      handle.serviceHandle = service.handle
      handle.charHandle = char.handle
      device.handles[handleInfo.handle] = handle
      lastHandle = handleInfo.handle
    }

  }

  device.handleOffset = this.globalHandleOffset
  device.highestHandle = int(lastHandle) + device.handleOffset
  this.globalHandleOffset += int(lastHandle)


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

