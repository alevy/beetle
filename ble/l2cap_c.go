package ble

// #include <sys/types.h>
// #include <sys/socket.h>
import "C"

import (
  "os"
  "unsafe"
)

type ConnInfo struct {
  HCIHandle uint16
  DevClass  [3]uint8
}

func GetConnInfo(fd *os.File) *ConnInfo {
  ci := make([]byte, 5)
  socklen := C.socklen_t(5)

  C.getsockopt(C.int(fd.Fd()), C.int(SOL_L2CAP), C.int(L2CAP_CONNINFO),
    unsafe.Pointer(&ci[0]), &socklen);

  var result ConnInfo
  result.HCIHandle = uint16(ci[0]) + uint16(ci[1]) << 8
  for i := 0; i < 3; i++ {
    result.DevClass[i] = ci[i + 2]
  }
  return &result
}

