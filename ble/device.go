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

type Transaction struct {
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

type Device struct {
  addr          string
  fd            io.ReadWriteCloser
  handles       map[uint16]*Handle
  handleOffset  int
  highestHandle int

  // Responses for client initiated transactions stream
  clientRespChan chan Response

  // Server initiated transactions
  serverReqChan  chan Request

  writeChan      chan []byte
  transactChan   chan Transaction

  connInfo       *ConnInfo
  first          bool
}

func (device *Device) String() string {
  return fmt.Sprintf("%s\t%d", device.addr, device.handleOffset)
}

func (device *Device) StrHandles() string {
  result := ""
  for i, handle := range device.handles {
    result += fmt.Sprintf(
      "0x%02X\t0x%02X:\t%v\t%v\t0x%02X\t0x%02X\tsubscribers: %d\n", i,
      handle.handle, handle.uuid, handle.cachedValue,
      handle.charHandle, handle.serviceHandle, len(handle.subscribers))
  }
  return result
}

func NewDevice(addr string, serverReqChan chan Request, 
  fd io.ReadWriteCloser, ci *ConnInfo) *Device {
  return &Device{addr, fd, make(map[uint16]*Handle), -1, -1,
    make(chan Response), serverReqChan,
    make(chan []byte), make(chan Transaction), ci, true}
}

func (this *Device) Disconnect() {
  close(this.clientRespChan)
  close(this.writeChan)
  close(this.transactChan)
  this.fd.Close()
}

func (this *Device) Start() {

  // Pull packets off `writeChan` and write to socket
  go func() {
    for req := range this.writeChan {
      if Debug {
        fmt.Printf("%s <= %v\n", this.addr, req)
      }
      this.fd.Write(req)
    }
  }()

  go func() {
    for req := range this.transactChan {
      if Debug {
        fmt.Printf("%s <- %v\n", this.addr, req)
      }
      this.writeChan <- req.packet
      resp :=<-this.clientRespChan
      req.respChan <-resp
    }
  }()

  // Read from socket and route to appropriate handler
  go func() {
    for {
      buf := make([]byte, 64)
      n, err := this.fd.Read(buf)
      if err != nil || n == 0 {
        return
      }

      buf = buf[0:n]
      if Debug {
        fmt.Printf("%s -> %v\n", this.addr, buf)
      }
      
      if (buf[0] & 1 == 1 && 
        buf[0] != ATT_OPCODE_HANDLE_VALUE_NOTIFICATION &&
        buf[0] != ATT_OPCODE_HANDLE_VALUE_INDICATION) ||
        buf[0] == ATT_OPCODE_HANDLE_VALUE_CONFIRMATION { 
        this.clientRespChan <-Response{buf, nil} // Response packet
      } else {
        this.serverReqChan <-Request{buf, this}
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
  go func() {
    respChan := make(chan Response)
    this.transactChan <-Transaction{packet,respChan}
    resp :=<-respChan
    cb(resp.value, resp.err)
  }()
}

