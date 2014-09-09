package main

import (
  "os"
)

const (
  HANDLE_FORMAT_16_BIT  uint8 = 1
  HANDLE_FORMAT_128_BIT uint8 = 2
)

type HandleInfo struct {
  format uint8
  handle uint16
  uuid UUID
}

type Proxy struct {
  fd *os.File
  handles map[uint16]HandleInfo
  nextRequest chan []byte
  reqChan chan []byte
  callbacks map[uint8](func([]byte))
}

func (this *Proxy) Init() {
  go this.requestThread()
}

func (this *Proxy) requestThread() {
  buf := make([]byte, 48)
  n, _ := this.fd.Read(buf)
  buf = buf[0:n]
  this.reqChan <- buf
}

func (this *Proxy) processNext() error {
  buf := make([]byte, 48)
  n, _ := this.fd.Read(buf)
  buf = buf[0:n]

  switch buf[0] {
  case ATT_OPCODE_FIND_INFO_RESPONSE:
    resp, err := NewFindInfoResponse(buf)
    if err != nil {
      return err
    }
    infoData := resp.InfoData()
    for _,infoDatum := range infoData {
      this.handles[infoDatum.handle] = infoDatum
    }
  }
  return nil
}

func (this *Proxy) FindInfo() {
  pkt := make([]byte, 5)
  pkt[0] = ATT_OPCODE_FIND_INFO_REQUEST

  var startHandle uint16 = 1
  var endHandle uint16 = 0xffff
  pkt[1] = uint8(startHandle); pkt[2] = uint8(startHandle >> 16)
  pkt[3] = uint8(endHandle); pkt[4] = uint8(endHandle >> 16)

  this.nextRequest <- pkt
}

