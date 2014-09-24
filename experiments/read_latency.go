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

  err = ble.HCIConnUpdate(hci, ci.HCIHandle, 40, 56, 0, 42)
  if err != nil {
    fmt.Printf("%s\n", err)
    os.Exit(1)
  }

  buf := make([]byte, 1024)
  minDuration := time.Hour
  var maxDuration time.Duration = 0
  var totalDuration time.Duration = 0
  req := []byte{ 0x0A, 0x06, 0x00 }

  for i := 0; i < 100; i++ {
    start := time.Now()
    f.Write(req)
    n, err := f.Read(buf)
    duration := time.Since(start)

    totalDuration += duration
    if minDuration > duration {
      minDuration = duration
    }
    if maxDuration < duration {
      maxDuration = duration
    }
    if err != nil {
      fmt.Printf("%s\n", err)
      os.Exit(1)
    }
    resp := buf[0:n]
    fmt.Printf("%d %v\n", duration / time.Millisecond, resp)
    time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
  }

  avgDuration := totalDuration / 100
  fmt.Printf("min: %d, max: %d, avg: %d\n", minDuration / time.Millisecond,
    maxDuration / time.Millisecond, avgDuration / time.Millisecond)
}

