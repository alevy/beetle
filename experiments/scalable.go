package main

import (
  "net"
  "fmt"
  "os"
  "time"
  "../ble"
  "sync/atomic"
)

func main() {
  runWorkers(1)
}

func runWorkers(num int) {
  for i := 0; i < num; i++ {
    go runWorker(uint16(i * 2 + 3))
    go runWorker(uint16(i * 2 + 1))
  }

  tick := time.Tick(1 * time.Second)
  for {
    <-tick
    c := atomic.SwapInt32(&count, 0)
    fmt.Printf("%d\n",c)
  }

}

var count int32 = 0

func runWorker(handle uint16) {
  remoteAddr, _ := net.ResolveTCPAddr("tcp", "localhost:5555")
  writeChan := make(chan []byte)

  conn, err := net.DialTCP("tcp", nil, remoteAddr)
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  go func() {
    for {
      buf := make([]byte, 48)
      n, err := conn.Read(buf)
      if err != nil {
        fmt.Printf("%s\n", err)
        os.Exit(1)
      }
      pkt := buf[0:n]
      switch pkt[0] {
      case ble.ATT_OPCODE_WRITE_REQUEST:
        writeChan <- []byte{ble.ATT_OPCODE_WRITE_RESPONSE}
      case ble.ATT_OPCODE_FIND_INFO_REQUEST:
        if pkt[1] <= 1 && pkt[2] == 0 {
          writeChan <- []byte{ble.ATT_OPCODE_FIND_INFO_RESPONSE, 0x2, 0x1, 0x0, 0x12, 0x34}
        } else {
          writeChan <- []byte{ble.ATT_OPCODE_ERROR, pkt[0], pkt[1], pkt[2], 0x0A}
          time.Sleep(1 * time.Second)
          writeChan <-[]byte{ble.ATT_OPCODE_FIND_INFO_REQUEST, 0x1, 0x0, 0xff, 0xff }
        }
      case ble.ATT_OPCODE_WRITE_RESPONSE:
        atomic.AddInt32(&count, 1)
        writeChan <-[]byte{ 0x12, uint8(handle), uint8(handle >> 8), 0x01 }
      case ble.ATT_OPCODE_FIND_INFO_RESPONSE:
        handle = uint16(pkt[2]) + uint16(pkt[3]) << 8
        writeChan <-[]byte{ 0x12, uint8(handle), uint8(handle >> 8), 0x01 }
      default:
        fmt.Printf("%v\n", pkt)
      }
    }
  }()

  go func() {

    for {
      req :=<-writeChan
      _, err := conn.Write(req)
      if err != nil {
        fmt.Printf("%s\n", err)
        os.Exit(1)
      }
    }
  }()
}

