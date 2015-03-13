package main

import (
  "runtime"
  "net"
  "fmt"
  "os"
  "time"
)

func main() {
  runWorkers(1)
  for i := 20; i < 200; i += 20 {
    runWorkers(i)
  }
}

func runWorkers(num int) {
  buf := make([]byte, 48)
  req := []byte{ 0x0A, 0x06, 0x00 }

  remoteAddr, _ := net.ResolveTCPAddr("tcp", "localhost:5555")

  done := false

  resultChan := make(chan int64, num * 100)
  doneChans := make([]chan bool, num)

  for i := 0; i < num; i++ {
    doneChan := make(chan bool)
    doneChans[i] = doneChan
    go func() {
      conn, err := net.DialTCP("tcp", nil, remoteAddr)
      if err != nil {
        fmt.Printf("%s\n", err)
        os.Exit(1)
      }

      for !done {
        start := time.Now()
        _, err := conn.Write(req)
        if err != nil {
          fmt.Printf("%s\n", err)
          os.Exit(1)
        }

        _, err = conn.Read(buf)
        runtime.Gosched()
        if err != nil {
          fmt.Printf("%s\n", err)
          os.Exit(1)
        }
        end := time.Now()
        l := end.UnixNano() / 1000000 - start.UnixNano() / 1000000
        resultChan <- l
      }
      doneChan <-true
    }()
  }


  tick := time.Tick(20 * time.Second)
  count := 0
  for {
    select {
    case l := <-resultChan:
      fmt.Printf("%d,%d\n", num, l)
      count += 1
    case <-tick:
      done = true
      for _,d := range doneChans {
        <-d
      }
      fmt.Printf("---%d,%f\n", num, float64(count) / 10)
      time.Sleep(1 * time.Second)
      return
    }
  }
}

