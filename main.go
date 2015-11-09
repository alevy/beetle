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
    PrintError(err)
    return // should probably exit
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
      PrintError(err)
    } else if len(lineBs) == 0 {
      continue
    }

    line := string(lineBs)
    parts := strings.Split(line, " ")

    switch parts[0] {
	    case "set-interval":
	      setInterval(manager, parts)
	    case "discover":
	    	discover(manager)
	    case "connect":
	      connect(manager, parts)
	    case "connectTCP":
	     	connectTCP(manager, parts)
	    case "disconnect":
	    	disconnect(manager, parts)
	    case "start":
	      start(manager, parts)
	    case "startnd":
	      startNd(manager, parts)
	    case "devices":
	    	devices(manager)
	    case "handles":
	      handles(manager, parts)
	    case "debug":
	    	debug(parts)
	    default:
	      fmt.Printf("Unknown command \"%s\"\n", parts[0])
    }
  }
}

func setInterval(manager *ble.Manager, args []string) {
  if len(args) < 2 {
   	PrintUsage("set-interval [interval]")
    return
  }

  interval64, err := strconv.ParseInt(args[1], 10, 0)
  if err != nil {
    PrintError(err)
    return
  }

  interval := uint16(interval64)
  for _,device := range(manager.Devices) {
    res := manager.ConnUpdate(device, interval)
    if res != 0 {
      fmt.Printf("ERROR: %d\n", err)
//      break
// TODO: break might be a problem if half of the devices get the new interval
    }
  }
}

func discover(manager *ble.Manager) {
	// TODO: implement
}

func connect(manager *ble.Manager, args []string) {
  if len(args) < 3 {
    PrintUsage("connect public|random DEVICE_ADDRESS [NICK]")
    return
  }
  
  addrType := args[1]
	address := args[2]

	nick := address 

	if len(args) >= 4 {
	  nick = args[3]
	}

	var err error
	switch addrType {
		case "public":
		  fmt.Printf("Connecting to %s... ", address)
		  err = manager.ConnectTo(ble.BDADDR_LE_PUBLIC, address, nick)
		case "random":
		  fmt.Printf("Connecting to %s... ", address)
		  err = manager.ConnectTo(ble.BDADDR_LE_RANDOM, address, nick)
		default:
		  fmt.Printf("Usage: connect public|private [device_address]\n")
		  return
	}

	if err != nil {
	  PrintError(err)
	} else {
	  fmt.Printf("done\n")
	}
}

func connectTCP(manager *ble.Manager, args []string) {
	if len(args) < 2 {
	  PrintUsage("connect IP:PORT [NICK]")
	  return
	}
	
	address := args[1]
	nick := "tcp://" + address
	if len(args) >= 3 {
	  nick = args[2]
	}

	fmt.Printf("Connecting to %s... ", address)
	err := manager.ConnectTCP(address, nick)
	if err != nil {
	  PrintError(err)
	} else {
	  fmt.Printf("done\n")
	}
}

func disconnect(manager *ble.Manager, args []string) {
  if len(args) < 2 {
    PrintUsage("disconnect [device_address]")
    return
  }

  address := args[1]

  fmt.Printf("Disconnecting from %s... ", address)
  err := manager.DisconnectFrom(address)

  if err != nil {
    PrintError(err)
  } else {
    fmt.Printf("done\n")
  }
}

func start(manager *ble.Manager, args []string) {
	if len(args) < 2 {
    PrintUsage("start [device_address]")
    return
  }

  address := args[1]
  fmt.Printf("Starting %s... ", address)
  
  // TODO: this doesn't look like it was checking for anything 
  // if err != nil {
  //   PrintError(err)
  //   return
  // }

  err := manager.Start(address)
  if err != nil {
    PrintError(err)
  } else {
    fmt.Printf("done\n")
  }
}

func startNd(manager *ble.Manager, args []string) {
	if len(args) < 2 {
    PrintUsage("startnd [device_address]")
    return
  }

 	address := args[1]
  fmt.Printf("Starting %s... ", address)

  // TODO: this doesn't look like it was checking for anything 
  // if err != nil {
  //   PrintError(err);
  //   return
  // }

  err := manager.StartNoDiscover(address)
  if err != nil {
    PrintError(err)
  } else {
    fmt.Printf("done\n")
  }
}

func devices(manager *ble.Manager) {
  if len(manager.Devices) == 0 {
    fmt.Printf("No connected devices\n")
  }
  for nick,device := range(manager.Devices) {
    fmt.Printf("%s:\t%s\n", nick, device)
  }
}

func handles(manager *ble.Manager, args []string) {
	if len(args) < 2 {
    PrintUsage("handles [device_nick]")
    return
  }

  address := args[1]

  device, ok := manager.Devices[address]
  if !ok {
    fmt.Printf("Unknown device %s\n", address)
    return
  }

  fmt.Printf("%s", device.StrHandles())
}

func debug(args []string) {
  if (len(args) < 2) {
    PrintUsage("debug on|off")
    return
  }

  if args[1] == "on" {
    ble.Debug = true
    fmt.Printf("Debugging on...\n")
  } else {
    fmt.Printf("Debugging off...\n")
    ble.Debug = false
  }
}

// prints an error
func PrintError(err error) {
	fmt.Printf("ERROR: %s\n", err)
}

// prints usage
func PrintUsage(usage string) {
	fmt.Printf("Usage: %s\n", usage)
}