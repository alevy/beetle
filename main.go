package main

import (
  "./ble"
  "bufio"
  "fmt"
  "io"
  "net"
  "os"
  "strconv"
  "strings"
)

func listenUnix(manager *ble.Manager) {
  /*os.Remove("/tmp/babel.sock")
  addr, _ := net.ResolveUnixAddr("unixpacket", "/tmp/babel.sock")
  listener, err := net.ListenUnix("unixpacket", addr)*/
  addr, _ := net.ResolveTCPAddr("tcp", ":5556")
  listener, err := net.ListenTCP("tcp", addr)
  if err != nil {
    fmt.Printf("ERROR: %s\n", err)
    os.Exit(1)
  }
  if err != nil {
    fmt.Printf("ERROR: %s\n", err)
    os.Exit(1)
  }

  for {
    conn, err := listener.Accept()
    if err != nil {
      fmt.Printf("ERROR: %s\n", err)
      os.Exit(1)
    }
    device := manager.AddDeviceForConn(conn.RemoteAddr().String(), conn, nil)
    manager.StartDevice(device)
    fmt.Printf("New app connected...\n")
  }
}

func main() {

  bio := bufio.NewReader(os.Stdin)
  hciSock, err := ble.NewHCI(0)
  if err != nil {
    fmt.Printf("%s\n", err)
  }

  manager := ble.NewManager(hciSock)

  go manager.RunRouter()
  go listenUnix(manager)

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
        fmt.Printf("Usage: connect public|random [device_address]\n")
        continue
      }
      switch parts[1] {
        case "public":
          fmt.Printf("Connecting to %s... ", parts[2])
          err = manager.ConnectTo(ble.BDADDR_LE_PUBLIC, parts[2])
        case "random":
          fmt.Printf("Connecting to %s... ", parts[2])
          err = manager.ConnectTo(ble.BDADDR_LE_RANDOM, parts[2])
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
        fmt.Printf("Usage: connect address:port\n")
        continue
      }
      fmt.Printf("Connecting to %s... ", parts[1])
      err = manager.ConnectTCP(parts[1])
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
      idx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      err = manager.DisconnectFrom(int(idx))
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
      idx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      err = manager.Start(int(idx))
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
      idx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      err = manager.StartNoDiscover(int(idx))
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
      } else {
        fmt.Printf("done\n")
      }
    case "devices":
      if len(manager.Devices) == 0 {
        fmt.Printf("No connected devices\n")
      }
      for idx,device := range(manager.Devices) {
        fmt.Printf("%02d:\t%s\n", idx, device)
      }
    case "handles":
      if len(parts) < 2 {
        fmt.Printf("Usage: handles [device_idx]\n")
        continue
      }
      idx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      if int(idx) >= len(manager.Devices) || int(idx) < 0 {
        fmt.Printf("Unknown device %s\n", parts[1])
        continue
      }
      device := manager.Devices[idx]
      fmt.Printf("%s", device.StrHandles())
    case "serve":
      if len(parts) < 3 {
        fmt.Printf("Usage: serve [server_idx] [client_idx]\n")
        continue
      }
      serverIdx, err := strconv.ParseInt(parts[1], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
      clientIdx, err := strconv.ParseInt(parts[2], 10, 0)
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }

      err = manager.ServeTo(int(serverIdx), int(clientIdx))
      if err != nil {
        fmt.Printf("ERROR: %s\n", err)
        continue
      }
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

