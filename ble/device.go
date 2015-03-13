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
  HandleInfo
  cachedTime time.Time
  cachedMap  map[*Device]bool
  subscribers []*Device
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

  readPkt        chan []byte
  clientRespChan chan Response
  serverReqChan  chan ManagerRequest
  writeChan      chan []byte
  transactionQ   chan Transaction

  connInfo        *ConnInfo
  first          bool
}

func (device *Device) String() string {
  return fmt.Sprintf("%s\t%d", device.addr, device.handleOffset)
}

func (device *Device) StrHandles() string {
  result := ""
  for _, handle := range device.handles {
    result += fmt.Sprintf("0x%02X:\t%v\t%v\tsubscribers: %d\n",
      handle.handle, handle.uuid, handle.cachedValue,
      len(handle.subscribers))
  }
  return result
}

func NewDevice(addr string, serverReqChan chan ManagerRequest, fd io.ReadWriteCloser, ci *ConnInfo) *Device {
  return &Device{addr, fd, make(map[uint16]*Handle), -1,
    make(chan []byte, 1), make(chan Response, 1), serverReqChan,
    make(chan []byte), make(chan Transaction, 100), ci, true}
}

func (this *Device) Start() {
  go func() {
    for {
      buf := make([]byte, 64)
      n, err := this.fd.Read(buf)
      if err != nil {
        close(this.readPkt)
        close(this.clientRespChan)
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

