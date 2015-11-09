package ble

import (
  "errors"
  "time"
  "io"
  "net"
  "os"
)

type Request struct {
  msg []byte
  device *Device
}

type Manager struct {
  Devices map[string]*Device
  globalHandleOffset int
  requestChan chan Request
  hciSock *os.File
}

func NewManager(hciSock *os.File) (*Manager) {
  return &Manager{make(map[string]*Device, 0), 0,
    make(chan Request), hciSock}
}

func (this *Manager) ConnectTo(addrType uint8, addr string, nick string) error {
  remoteAddr, err := Str2Ba(addr)
  if err != nil {
    return err
  }

  f, err := NewBLE(NewL2Sockaddr(4, remoteAddr, addrType), addr)
  if err != nil {
    return err
  }

  ci := GetConnInfo(f)

  this.AddDeviceForConn(addr, nick, f, ci)

  return nil
}

func (this *Manager) ConnectTCP(addr string, nick string) (error) {
  conn, err := net.Dial("tcp", addr)
  if err != nil {
    return err
  }
  this.AddDeviceForConn("tcp://" + addr, nick, conn, nil)
  return nil
}


func (this *Manager) ConnUpdate(device *Device, interval uint16) int {
  if device.connInfo != nil {
    return HCIConnUpdate(this.hciSock, device.connInfo.HCIHandle, interval, interval, 0, 0x0C80)
  } else {
    return 0
  }
}

func (this *Manager) AddDeviceForConn(addr string, nick string,
                            f io.ReadWriteCloser, ci *ConnInfo) (*Device) {
  device := NewDevice(addr, this.requestChan, f, ci)
  this.Devices[nick] = device
  return device
}

func (this *Manager) StartNoDiscover(nick string) error {
  device, ok := this.Devices[nick]
  if ok {
    device.Start()
    return nil
  } else {
    return errors.New("No such device")
  }
}

func (this *Manager) Start(nick string) error {
  device, ok := this.Devices[nick]
  if ok {
    return this.StartDevice(device)
  } else {
    return errors.New("No such device")
  }
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
    handle.subscribers = make(map[*Device]bool)
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
      handle.subscribers = make(map[*Device]bool)
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
        handle.subscribers = make(map[*Device]bool)
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
      handle.subscribers = make(map[*Device]bool)
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

func (this *Manager) DisconnectFrom(nick string) error {
  device, ok := this.Devices[nick]
  if !ok {
    return errors.New("No such device")
  }

  device.Disconnect()

  delete(this.Devices, nick)

  // TODO(alevy): This is really really inefficient. Structuring subscriptions
  // better would make this easier. For our purposes at the moment, 10s of
  // devices with 10s of handles, so the iteration is probably not so bad. At
  // the limit, this could be 16 thousand iterations for each device, which is
  // a lot.
  for _,d := range this.Devices {
    for _, handle := range d.handles {
      if _, ok := handle.subscribers[device]; ok {
        delete(handle.subscribers, device)
        if len(handle.subscribers) == 0 {
          // This was the last subscriber, so unsubscribe.

          // Iterate through characteristic to find a client configuration
          char := d.handles[handle.charHandle]
          for i := handle.charHandle; i <= char.endGroup; i++ {
            handle, ok = d.handles[i]
            if !ok {
              break
            }

            if handle.uuid == GATT_CLIENT_CONFIGURATION_UUID {
              // adjust handle offset
              h := i - uint16(d.handleOffset)
              // write a zero
              d.Transaction(
                []byte{ATT_OPCODE_WRITE_REQUEST, byte(h & 0xff), 
                  byte(h >> 8), 0, 0},
                func(resp []byte, err error){});
              break
            }
          }
        }
      }
    }
  }

  return nil
}

