package main

import (
  "bufio"
  "errors"
  "fmt"
  "io"
  "os"
  "strconv"
  "strings"
)

type Manager struct {
  devices []*Device
  globalHandleOffset int
  associations map[int][]int
}

func (this *Manager) connectTo(addr string) error {
  remoteAddr, err := Str2Ba(addr)
  if err != nil {
    return err
  }

  f, err := NewBLE(NewL2Sockaddr(4, remoteAddr, BDADDR_LE_RANDOM), addr)
  if err != nil {
    return err
  }

  device := NewDevice(addr, f)
  this.devices = append(this.devices, device)

  return nil
}

func (this *Manager) start(idx int) error {
  if idx >= len(this.devices) || idx < 0 {
    return errors.New("No such device")
  }

  device := this.devices[idx]
  device.Start()
  device.StartClient()

  handles, err := DiscoverHandles(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  this.globalHandleOffset += len(handles) + 1
  device.handleOffset = this.globalHandleOffset

  for _, handle := range handles {
    device.handles[handle.handle] = handle
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

  go device.StartServer()

  return nil
}

func (this *Manager) disconnectFrom(idx int) error {
  if idx >= len(this.devices) || idx < 0 {
    return errors.New("No such device")
  }

  device := this.devices[idx]

  err := device.fd.Close()
  if err != nil {
    return err
  }

  this.devices = append(this.devices[0:idx], this.devices[idx + 1:]...)
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

func (this *Manager) serveTo(serverIdx, clientIdx int) error {
  if clientIdx >= len(this.devices) || serverIdx >= len(this.devices) {
    return errors.New("No such device")
  }

  if lst,ok := this.associations[clientIdx]; ok {
    this.associations[clientIdx] = append(lst, serverIdx)
  } else {
    this.associations[clientIdx] = []int {serverIdx}
  }

  return nil
}

func main() {

  bio := bufio.NewReader(os.Stdin)
  manager := &Manager{make([]*Device, 0), 0, make(map[int][]int)}
  for {
    fmt.Printf("> ")
    lineBs, _, err := bio.ReadLine()
    if err == io.EOF {
      fmt.Printf("\nQuitting...\n")
      break
    } else if err != nil {
      fmt.Printf("ERROR: %s\n", err)
    } else if len(lineBs) == 0 {
      continue
    }

    line := string(lineBs)
    parts := strings.Split(line, " ")

    switch parts[0] {
    case "connect":
      if len(parts) < 2 {
        fmt.Printf("Usage: connect [device_address]\n")
        continue
      }
      fmt.Printf("Connecting to %s... ", parts[1])
      err = manager.connectTo(parts[1])
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "disconnect":
      if len(parts) < 2 {
        fmt.Printf("Usage: disconnect [device_address]\n")
        continue
      }
      fmt.Printf("Disconnecting from %s... ", parts[1])
      idx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      err = manager.disconnectFrom(int(idx))
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "start":
      if len(parts) < 2 {
        fmt.Printf("Usage: start [device_address]\n")
        continue
      }
      fmt.Printf("Starting %s... ", parts[1])
      idx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      err = manager.start(int(idx))
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "devices":
      if len(manager.devices) == 0 {
        fmt.Printf("No connected devices\n")
      }
      for idx,device := range(manager.devices) {
        fmt.Printf("%02d:\t%s\n", idx, device.addr)
      }
    case "handles":
      if len(parts) < 2 {
        fmt.Printf("Usage: handles [device_idx]\n")
        continue
      }
      idx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      if int(idx) >= len(manager.devices) || int(idx) < 0 {
        fmt.Printf("Unknown device %s\n", parts[1])
        continue
      }
      device := manager.devices[idx]
      for _, handle := range device.handles {
        fmt.Printf("0x%02X:\t%v\t%v\n",
          handle.handle, handle.uuid, handle.cachedValue)
      }
    case "serve":
      if len(parts) < 3 {
        fmt.Printf("Usage: serve [server_idx] [client_idx]\n")
      }
      serverIdx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      clientIdx, err := strconv.ParseInt(parts[2], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }

      err = manager.serveTo(int(serverIdx), int(clientIdx))
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
    default:
      fmt.Printf("Unknown command \"%s\"\n", parts[0])
    }
  }
}

