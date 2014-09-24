package ble

import (
  "fmt"
  "syscall"
  "os"
  "unsafe"
)

func NewHCI(dev_id uint8) (*os.File, error) {
  fd, err := syscall.Socket(AF_BLUETOOTH, syscall.SOCK_RAW | syscall.SOCK_CLOEXEC,
    BTPROTO_HCI)
  if err != nil {
    return nil, err
  }

  var sockaddr_hci [4]uint8
  *((*uint16)(unsafe.Pointer(&sockaddr_hci[0]))) = uint16(AF_BLUETOOTH)
  sockaddr_hci[2] = dev_id


  addrlen := len(sockaddr_hci)
  _, _, err1 := syscall.Syscall(syscall.SYS_BIND, uintptr(fd),
      uintptr(unsafe.Pointer(&sockaddr_hci[0])), uintptr(addrlen))
  if err1 != 0 {
    return nil, err
  }

  f := os.NewFile(uintptr(fd), "hci")
  return f, nil
}

func HCIConnUpdate(fd *os.File, handle, min_interval, max_interval,
  latency, supervisor_timeout uint16) (error) {
    // HCI_OP_LE_CONN_UPDATE		0x2013
    // size 14

    buf := make([]byte, 17)
    buf[0] = 0x13
    buf[1] = 0x20
    buf[2] = 14

    // handle
    buf[3] = uint8(handle)
    buf[4] = uint8(handle >> 8)

    // conn interval min/max
    buf[5] = uint8(min_interval)
    buf[6] = uint8(min_interval >> 8)
    buf[7] = uint8(max_interval)
    buf[8] = uint8(max_interval >> 8)

    // latency
    buf[9] = uint8(latency)
    buf[10] = uint8(latency >> 8)

    // supervisor timeout
    buf[11] = uint8(supervisor_timeout)
    buf[12] = uint8(supervisor_timeout >> 8)

    buf[13] = 0
    buf[14] = 0
    buf[15] = 0
    buf[16] = 0

    fmt.Printf("CONN UDPATE %v\n", buf)

    _, err := fd.Write(buf)
    return err
}

