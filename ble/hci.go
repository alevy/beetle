package ble

// #include <bluetooth/bluetooth.h>
// #include <bluetooth/hci.h>
// #include <bluetooth/hci_lib.h>
// #cgo LDFLAGS: -lbluetooth
import "C"

import (
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
  latency, supervisor_timeout uint16) (int) {
    err := C.hci_le_conn_update(C.int(fd.Fd()), C.uint16_t(handle),
      C.uint16_t(min_interval), C.uint16_t(max_interval), C.uint16_t(latency), C.uint16_t(supervisor_timeout), 0)
    return int(err)
}

