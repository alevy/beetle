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

type subscriptionRequest struct {
  opcode  uint8
  channel chan []byte
}

type Device struct {
  addr          string
  fd            *os.File
  handles       map[uint16]*HandleInfo
  handleOffset  uint16
  subscriptions map[uint8](chan []byte)
  subscribeChan chan subscriptionRequest
}

func (this *Device) ProcessIncoming() error {
  buf := make([]byte, 48)

  for {
    sReq, ok := <-this.subscribeChan
    if !ok {
      return nil
    }
    this.subscriptions[sReq.opcode] = sReq.channel

    n, err := this.fd.Read(buf)
    if err != nil {
      return err
    }

    req := buf[0:n]
    var opcode uint8
    if req[0] == ATT_OPCODE_ERROR {
      // Use requested opcode for filter
      opcode = req[1] + 1 // request always one less than response
    } else {
      opcode = req[0]
    }

    ch := this.subscriptions[opcode]
    if ch != nil {
      ch <-req
      delete(this.subscriptions, opcode)
    } else {
      for {
        sReq, ok := <-this.subscribeChan
        if !ok {
          return nil
        }

        if sReq.opcode == opcode {
          sReq.channel <-req
          break
        } else {
          this.subscriptions[sReq.opcode] = sReq.channel
        }
      }
    }
  }
}

type TransactionReadWriter struct {
  device *Device
  channel chan []byte
}

func (this *TransactionReadWriter) Read(buf []byte) (int, error) {
  recv := <-this.channel
  copy(buf, recv)
  return len(recv), nil
}

func (this *TransactionReadWriter) Write(buf []byte) (int, error) {
  this.channel = make(chan []byte)
  opcode := buf[0] + 1
  this.device.subscribeChan <-subscriptionRequest{opcode, this.channel}
  return this.device.fd.Write(buf)
}

type Manager struct {
  devices []*Device
  globalHandleOffset uint16
  associations map[int][]int
}

func (this *Manager) connectTo(addr string) error {
  remoteAddr, err := Str2Ba(addr)
  if err != nil {
    return err
  }

  f, err := NewBLE(NewL2Sockaddr(4, remoteAddr, BDADDR_LE_RANDOM))
  if err != nil {
    return err
  }

  device := &Device{addr, f, make(map[uint16]*HandleInfo), this.globalHandleOffset,
              make(map[uint8](chan []byte)), make(chan subscriptionRequest)}
  this.devices = append(this.devices, device)

  go device.ProcessIncoming()

  handles, err := DiscoverHandles(
    &TransactionReadWriter{device, nil})
  if err != nil {
    f.Close()
    return err
  }

  for _, handle := range handles {
    device.handles[handle.handle] = handle
  }

  groupVals, err := DiscoverServices(
    &TransactionReadWriter{device, nil})
  if err != nil {
    f.Close()
    return err
  }

  for _,v := range groupVals {
    device.handles[v.handle].cachedValue = v.value
  }

  this.globalHandleOffset += uint16(len(handles) + 1)


  return nil
}

func (this *Manager) disconnectFrom(idx int) error {
  if idx >= len(this.devices) {
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
      device := manager.devices[idx]
      if device == nil {
        fmt.Printf("Unknown device %s\n", parts[1])
      }
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

