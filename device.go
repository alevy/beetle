package main

import (
  "os"
)

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
  handleOffset  int32

  clientRespChan chan Response
  serverRespChan chan Response
  reqChan        chan []byte

  clientInChan    chan WriteReq
}

func NewDevice(addr string, fd *os.File) *Device {
  return &Device{addr, fd, make(map[uint16]*HandleInfo), -1,
    make(chan Response), make(chan Response), make(chan []byte),
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
        this.serverRespChan <-Response{nil, err}
      } else {
        resp := buf[0:n]
        if (buf[0] & 1 == 1 && buf[0] != ATT_OPCODE_HANDLE_VALUE_NOTIFICATION &&
            buf[0] != ATT_OPCODE_HANDLE_VALUE_INDICATION) ||
            buf[0] == ATT_OPCODE_HANDLE_VALUE_CONFIRMATION { // Response packet
          this.clientRespChan <-Response{resp, nil}
        } else {
          this.serverRespChan <-Response{resp, nil}
        }
      }
    }
  }()

  // Pass write packets to socket
  go func() {
    for {
      req := <-this.reqChan
      this.fd.Write(req)
    }
  }()

  // Client-side loop
  go func() {
    for {
      req := <-this.clientInChan
      this.reqChan <-req.packet
      if req.respChan != nil {
        resp := <-this.clientRespChan
        req.respChan <-resp
      }
    }
  }()

  //TODO: Server-side loop
}

func (this *Device) WriteCmd(packet []byte) {
  this.clientInChan <- WriteReq{packet, nil}
}

func (this *Device) Transaction(packet []byte) ([]byte, error) {
  outChan := make(chan Response)
  this.clientInChan <-WriteReq{packet, outChan}
  resp := <-outChan
  return resp.value, resp.err
}

