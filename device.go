package main

import (
  "fmt"
  "os"
)

var debug bool = false

type Response struct {
  value []byte
  err   error
}

type WriteReq struct {
  packet        []byte
  respChan      chan Response
}

type Device struct {
  addr          string
  fd            *os.File
  handles       map[uint16]*HandleInfo
  handleOffset  int

  clientRespChan chan Response
  serverReqChan  chan ManagerRequest
  writeChan      chan []byte

  clientInChan    chan WriteReq
}

func NewDevice(addr string, serverReqChan chan ManagerRequest, fd *os.File) *Device {
  return &Device{addr, fd, make(map[uint16]*HandleInfo), -1,
    make(chan Response, 1), serverReqChan, make(chan []byte),
    make(chan WriteReq)}
}

func (this *Device) Start() {
  // Read from socket and route to appropriate handler
  go func() {
    for {
      buf := make([]byte, 64)
      n, err := this.fd.Read(buf)
      if err != nil {
        this.clientRespChan <-Response{nil, err}
      } else {
        if debug {
          fmt.Printf("%s -> %v\n", this.addr, buf[0:n])
        }
        resp := buf[0:n]
        if (buf[0] & 1 == 1 && buf[0] != ATT_OPCODE_HANDLE_VALUE_NOTIFICATION &&
            buf[0] != ATT_OPCODE_HANDLE_VALUE_INDICATION) ||
            buf[0] == ATT_OPCODE_HANDLE_VALUE_CONFIRMATION { // Response packet
          this.clientRespChan <-Response{resp, nil}
        } else {
          this.serverReqChan <-ManagerRequest{resp, this}
        }
      }
    }
  }()

  // Pass write packets to socket
  go func() {
    for {
      req := <-this.writeChan
      if debug {
        fmt.Printf("%s <- %v\n", this.addr, req)
      }
      this.fd.Write(req)
    }
  }()
}

func (this *Device) StartClient() {
  // Client-side loop
  go func() {
    for {
      req := <-this.clientInChan
      this.writeChan <-req.packet
      if req.respChan != nil {
        resp := <-this.clientRespChan
        req.respChan <-resp
      }
    }
  }()
}

/*func (this *Device) StartServer() {
  //TODO: Server-side loop
  for {
    req := <-this.serverReqChan
    if req.err != nil {
    } else {
      this.writeChan <- []byte{1, req.value[0], 0, 0, 0x11}
    }
  }
}*/

func (this *Device) WriteCmd(packet []byte) {
  this.clientInChan <- WriteReq{packet, nil}
}

func (this *Device) Respond(packet []byte) {
  this.writeChan <-packet
}

func (this *Device) Transaction(packet []byte) ([]byte, error) {
  outChan := make(chan Response)
  this.clientInChan <-WriteReq{packet, outChan}
  resp := <-outChan
  return resp.value, resp.err
}

