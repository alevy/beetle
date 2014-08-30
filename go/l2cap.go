package main

import (
  "errors"
  "encoding/binary"
  "fmt"
  "strconv"
  "strings"
  "syscall"
  "os"
  "unsafe"
)

const AF_BLUETOOTH int = 31
const (
  BTPROTO_L2CAP int = 0
  BTPROTO_HCI   int = 1
  BTPROTO_SCO   int = 2
  BTPROTO_RFCOM int = 3
  BTPROTO_BNEP  int = 4
  BTPROTO_CMTP  int = 5
  BTPROTO_HIDP  int = 6
  BTPROTO_AVDTP int = 7
)

const (
  SOL_HCI int = 0
  SOL_L2CAP int = 6
  SOL_SCO int = 17
  SOL_RFCOMM int = 18
)

const (
  BDADDR_BREDR     uint8 = 0x00
  BDADDR_LE_PUBLIC uint8 = 0x01
  BDADDR_LE_RANDOM uint8 = 0x02
)

const (
  L2CAP_LM int = 0x03
  L2CAP_LM_MASTER int =0x0001
  L2CAP_LM_AUTH int = 0x0002
  L2CAP_LM_ENCRYPT int = 0x0004
  L2CAP_LM_TRUSTED int = 0x0008
  L2CAP_LM_RELIABLE int = 0x0010
  L2CAP_LM_SECURE int = 0x0020
  L2CAP_LM_FIPS int = 0x0040
)

type L2Sockaddr struct {
  buf [13]uint8
}

func NewL2Sockaddr(channel uint16, addr [6]uint8, addr_type uint8) *L2Sockaddr {
  res := &L2Sockaddr{}
  *((*uint16)(unsafe.Pointer(&res.buf[0]))) = uint16(AF_BLUETOOTH)
  for i := 0; i < len(addr); i++ {
    res.buf[4 + i] = addr[i]
  }

  le_chan := make([]byte, 2)
  binary.LittleEndian.PutUint16(le_chan, channel)
  res.buf[10] = le_chan[0]
  res.buf[11] = le_chan[1]

  res.buf[12] = addr_type
  return res
}

func bind(s int, addr *L2Sockaddr) (err error) {
  addrlen := len(addr.buf)
  _, _, e1 := syscall.Syscall(syscall.SYS_BIND, uintptr(s),
      uintptr(unsafe.Pointer(&addr.buf[0])), uintptr(addrlen))
  if e1 != 0 {
    err = e1
  }
  return
}

func connect(s int, addr *L2Sockaddr) (err error) {
  addrlen := len(addr.buf)
  _, _, e1 := syscall.Syscall(syscall.SYS_CONNECT, uintptr(s),
      uintptr(unsafe.Pointer(&addr.buf[0])), uintptr(addrlen))
  if e1 != 0 {
    err = e1
  }
  return
}

func NewBLE(remoteAddr *L2Sockaddr) (*os.File, error){
  fd, err := syscall.Socket(AF_BLUETOOTH, syscall.SOCK_SEQPACKET, BTPROTO_L2CAP);
  if err != nil {
    return nil, err
  }

  addr := NewL2Sockaddr(4, [6]uint8{0, 0, 0, 0, 0, 0}, BDADDR_LE_PUBLIC)
  err = bind(fd, addr)
  if err != nil {
    return nil, err
  }

  opt, err := syscall.GetsockoptInt(fd, SOL_L2CAP, L2CAP_LM)
  if err != nil {
    return nil, err
  }

  err = syscall.SetsockoptInt(fd, SOL_L2CAP, L2CAP_LM, opt | L2CAP_LM_MASTER)
  if err != nil {
    return nil, err
  }

//  err = syscall.SetsockoptInt(fd, SOL_L2CAP, L2CAP_LM, L2CAP_LM_AUTH)
//  if err != nil {
//    return nil, err
//  }

  err = connect(fd, remoteAddr)
  if err != nil {
    return nil, err
  }

  return os.NewFile(uintptr(fd), "btle"), nil

}

func Str2Ba(addrStr string) ([6]uint8, error) {
  var remoteAddr [6]uint8
  addrComponents := strings.Split(addrStr, ":")
  if (len(addrComponents) != 6) {
    return remoteAddr, errors.New("Bad address format")
  }

  for i,c := range(addrComponents) {
    dig, err := strconv.ParseUint(c, 16, 8)
    remoteAddr[5 - i] = uint8(dig)
    if err != nil {
      return remoteAddr, err
    }
  }
  return remoteAddr, nil
}

func Proxy(self *os.File, remote *os.File) {
  buf := make([]byte, 48)
  for {
    n, err := self.Read(buf)
    if err != nil {
      fmt.Printf("%s\n", err)
    }
    fmt.Printf("%v\n", buf[0:n])
    _, err = remote.Write(buf[0:n])
    if err != nil {
      fmt.Printf("%s\n", err)
    }
  }
}

func main() {
  remoteAddr1, err := Str2Ba(os.Args[1])
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }
  remoteAddr2, err := Str2Ba(os.Args[2])
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  conn1, err := NewBLE(NewL2Sockaddr(4, remoteAddr1, BDADDR_LE_RANDOM))
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  conn2, err := NewBLE(NewL2Sockaddr(4, remoteAddr2, BDADDR_LE_RANDOM))
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  /*conn1.Write(FindInfoRequest(1, 0xffff))
  conn2.Write(FindInfoRequest(1, 0xffff))

  buf := make([]byte, 48)
  n,_ := conn1.Read(buf)
  fmt.Printf("%v\n", buf[0:n])

  n,_ = conn2.Read(buf)
  fmt.Printf("%v\n", buf[0:n])*/

  go Proxy(conn1, conn2)
  Proxy(conn2, conn1)

}

