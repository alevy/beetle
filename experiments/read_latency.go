package main


// Measures the latency to do a read from the desktop to a peripheral device.
// Goal: does a read return within a single connection interval or on the next
//       one? If the range is between 0 and 1 connection interval, yes, if
//       larger, no

import (
  "../ble"
  "fmt"
  "math/rand"
  "os"
  "time"
)

func main() {
  addr, err := ble.Str2Ba(os.Args[1])
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  f, err := ble.NewBLE(ble.NewL2Sockaddr(4, addr, ble.BDADDR_LE_RANDOM), os.Args[1])
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  ci := ble.GetConnInfo(f)
  hci, err := ble.NewHCI(0)
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  var interval uint16
  for interval = 6; interval < 0x0C80; interval *= 2 {
    errc := ble.HCIConnUpdate(hci, ci.HCIHandle, interval, interval, 0, 0x0C80)
    if errc != 0 {
      fmt.Printf("Failed to update %d\n", err)
      os.Exit(1)
    }

    buf := make([]byte, 1024)
    req := []byte{ 0x0A, 0x06, 0x00 }

    for i := 0; i < 30; i++ {
      start := time.Now()
      f.Write(req)
      _, err := f.Read(buf)
      duration := time.Since(start)

      if err != nil {
        fmt.Printf("%s\n", err)
        os.Exit(1)
      }
      fmt.Printf("%d,%d\n", interval, duration / time.Millisecond)
      time.Sleep(time.Duration(rand.Intn(int(float64(time.Duration(interval) * time.Millisecond) * 1.25))))
    }
  }
}

