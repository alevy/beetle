package main

import (
  "./ble"
  "bufio"
  "fmt"
  "io"
  "os"
  "strconv"
  "strings"
)

func main() {

  bio := bufio.NewReader(os.Stdin)
  hciSock, err := ble.NewHCI(0)
  if err != nil {
    fmt.Printf("%s\n", err)
  }

  manager := ble.NewManager(hciSock)

  go manager.RunRouter()

  for {
    fmt.Printf("> ")
    lineBs, _, err := bio.ReadLine()
    if err == io.EOF {
      fmt.Printf("\nQuitting...\n")
      break
    } else if err != nil {
      fmt.Printf("ERROR: %s\n", err)
    } else if len(lineBs) == 0 {
      continue
    }

    line := string(lineBs)
    parts := strings.Split(line, " ")

    switch parts[0] {
    case "panic":
      panic("AAA")
    case "set-interval":
      if len(parts) < 2 {
        fmt.Printf("Usage: set-interval [interval]\n")
        continue
      }
      interval64, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("%s\n", err)
        continue
      }
      interval := uint16(interval64)
      for _,device := range(manager.Devices) {
        res := manager.ConnUpdate(device, interval)
        if res != 0 {
          fmt.Printf("ERROR: %d\n", res)
          break
        }
      }
    case "connect":
      if len(parts) < 3 {
        fmt.Printf("Usage: connect public|random DEVICE_ADDRESS [NICK]\n")
        continue
      }
      addrType := parts[1]
      address := parts[2]
      nick := address
      if len(parts) >= 4 {
        nick = parts[3]
      }
      switch addrType {
        case "public":
          fmt.Printf("Connecting to %s... ", address)
          err = manager.ConnectTo(ble.BDADDR_LE_PUBLIC, address, nick)
        case "random":
          fmt.Printf("Connecting to %s... ", parts[2])
          err = manager.ConnectTo(ble.BDADDR_LE_RANDOM, address, nick)
        default:
          fmt.Printf("Usage: connect public|private [device_address]\n")
          continue
      }
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "connectTCP":
      if len(parts) < 2 {
        fmt.Printf("Usage: connect IP:PORT [NICK]\n")
        continue
      }
      address := parts[1]
      nick := "tcp://" + address
      if len(parts) >= 3 {
        nick = parts[2]
      }
      fmt.Printf("Connecting to %s... ", address)
      err = manager.ConnectTCP(address, nick)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "disconnect":
      if len(parts) < 2 {
        fmt.Printf("Usage: disconnect [device_address]\n")
        continue
      }
      fmt.Printf("Disconnecting from %s... ", parts[1])
      err = manager.DisconnectFrom(parts[1])
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "start":
      if len(parts) < 2 {
        fmt.Printf("Usage: start [device_address]\n")
        continue
      }
      fmt.Printf("Starting %s... ", parts[1])
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      err = manager.Start(parts[1])
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "startnd":
      if len(parts) < 2 {
        fmt.Printf("Usage: startnd [device_address]\n")
        continue
      }
      fmt.Printf("Starting %s... ", parts[1])
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      err = manager.StartNoDiscover(parts[1])
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "devices":
      if len(manager.Devices) == 0 {
        fmt.Printf("No connected devices\n")
      }
      for nick,device := range(manager.Devices) {
        fmt.Printf("%02d:\t%s\n", nick, device)
      }
    case "handles":
      if len(parts) < 2 {
        fmt.Printf("Usage: handles [device_nick]\n")
        continue
      }
      device, ok := manager.Devices[parts[1]]
      if !ok {
        fmt.Printf("Unknown device %s\n", parts[1])
        continue
      }
      fmt.Printf("%s", device.StrHandles())
    case "debug":
      if (len(parts) < 2) {
        fmt.Printf("Usage: debug on|off\n")
        continue
      }
      if parts[1] == "on" {
        ble.Debug = true
        fmt.Printf("Debugging on...\n")
      } else {
        fmt.Printf("Debugging off...\n")
        ble.Debug = false
      }
    default:
      fmt.Printf("Unknown command \"%s\"\n", parts[0])
    }
  }
}

