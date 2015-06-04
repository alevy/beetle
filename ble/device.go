package ble

import (
  "fmt"
  "io"
  "time"
)

var Debug bool = false

type Response struct {
  value []byte
  err   error
}

type WriteReq struct {
  packet        []byte
  respChan      chan Response
}

type Handle struct {
  handle uint16
  uuid UUID
  endGroup uint16
  cachedValue []byte
  cachedTime time.Time
  cachedMap  map[*Device]bool
  cachedInfinite bool
  serviceHandle uint16
  charHandle uint16
  subscribers map[*Device]bool
}

type Transaction struct {
  req   []byte
  cb    func([]byte, error)
}

type Device struct {
  addr          string
  fd            io.ReadWriteCloser
  handles       map[uint16]*Handle
  handleOffset  int
  highestHandle int

  readPkt        chan []byte
  clientRespChan chan Response
  serverReqChan  chan ManagerRequest
  writeChan      chan []byte

  connInfo       *ConnInfo
  first          bool
}

func (device *Device) String() string {
  return fmt.Sprintf("%s\t%d", device.addr, device.handleOffset)
}

func (device *Device) StrHandles() string {
  result := ""
  for i, handle := range device.handles {
    result += fmt.Sprintf("0x%02X\t0x%02X:\t%v\t%v\t0x%02X\t0x%02X\tsubscribers: %d\n",
      i,
      handle.handle, handle.uuid, handle.cachedValue,
      handle.charHandle, handle.serviceHandle, len(handle.subscribers))
  }
  return result
}

func NewDevice(addr string, serverReqChan chan ManagerRequest, fd io.ReadWriteCloser, ci *ConnInfo) *Device {
  return &Device{addr, fd, make(map[uint16]*Handle), -1, -1,
    make(chan []byte, 2), make(chan Response, 2), serverReqChan,
    make(chan []byte, 2), ci, true}
}

func (this *Device) Start() {
  go func() {
    for {
      buf := make([]byte, 64)
      if Debug {
        fmt.Printf("Reading from %s\n", this.addr)
      }
      n, err := this.fd.Read(buf)
      if Debug {
        fmt.Printf("Read from %s: %v\n", this.addr, buf[0:n])
      }
      if err != nil {
        return
      }
      this.readPkt <- buf[0:n]
    }
  }()

  // Pass write packets to socket
  go func() {
    for req := range this.writeChan {
      if Debug {
        fmt.Printf("%s <- %v\n", this.addr, req)
      }
      _, err := this.fd.Write(req)
      if Debug {
        fmt.Printf("Wrote to %s\n", this.addr)
      }
      if err != nil {
        return
      }
    }
  }()

  // Read from socket and route to appropriate handler
  go func() {
    for buf := range(this.readPkt) {
      if Debug {
        fmt.Printf("%s -> %v\n", this.addr, buf)
      }
      if (buf[0] & 1 == 1 && buf[0] != ATT_OPCODE_HANDLE_VALUE_NOTIFICATION &&
          buf[0] != ATT_OPCODE_HANDLE_VALUE_INDICATION) ||
          buf[0] == ATT_OPCODE_HANDLE_VALUE_CONFIRMATION { // Response packet
        this.clientRespChan <-Response{buf, nil}
      } else {
        this.serverReqChan <-ManagerRequest{buf, this}
      }
    }
  }()

}

func (this *Device) Respond(packet []byte) {
  this.writeChan <-packet
}

func (this *Device) WriteCmd(packet []byte) {
  this.writeChan <-packet
}

func (this *Device) Transaction(packet []byte, cb func([]byte, error)) {
  this.writeChan <-packet
  resp :=<-this.clientRespChan
  cb(resp.value, resp.err)
}

