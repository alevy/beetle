package main

import (
  "bytes"
  "errors"
  "sort"
)

type ManagerRequest struct {
  msg []byte
  device *Device
}

type Manager struct {
  devices []*Device
  globalHandleOffset int
  associations map[int][]int
  requestChan chan ManagerRequest
}

func NewManager() (*Manager) {
  return &Manager{make([]*Device, 0), 0, make(map[int][]int),
    make(chan ManagerRequest)}
}

func (this *Manager) RunRouter() {
  for req := range this.requestChan {
    pkt := req.msg
    switch(pkt[0]) {
    case ATT_OPCODE_FIND_BY_TYPE_VALUE_REQUEST:
      findReq, err := ParseFindByTypeValueRequest(pkt)
      if err != nil {
        resp := NewError(ATT_OPCODE_FIND_BY_TYPE_VALUE_REQUEST, 0, 4)
        req.device.Respond(resp.msg)
        continue
      }

      startHandle := findReq.StartHandle()
      endHandle := findReq.EndHandle()
      attType := findReq.Type()
      attVal := findReq.Value()

      handles := make(GroupValueLst, 0, 100)
      for _,device := range this.devices {
        if device == req.device || device.handleOffset < 0 {
          continue
        }
        if device.handleOffset > int(endHandle) {
          break
        }

        for _, handle := range device.handles {
          if handle.uuid == attType && bytes.Equal(handle.cachedValue, attVal) {
            offset := uint16(device.handleOffset)
            h := &GroupValue{handle.handle + offset, handle.endGroup + offset, nil}
            handles = append(handles, h)
          }
        }
      }

      sort.Sort(handles)

      if len(handles) > 0 {
        resp := NewFindByTypeValueResponse(handles)
        req.device.Respond(resp.msg)
      } else {
        resp := NewError(ATT_OPCODE_FIND_BY_TYPE_VALUE_REQUEST, startHandle, 0x0A)
        req.device.Respond(resp.msg)
      }
    case ATT_OPCODE_READ_BY_TYPE_REQUEST:
      readReq, err := ParseReadByTypeRequest(pkt)
      if err != nil {
        resp := NewError(ATT_OPCODE_READ_BY_TYPE_REQUEST, 0, 4)
        req.device.Respond(resp.msg)
        continue
      }

      startHandle := readReq.StartHandle()
      endHandle := readReq.EndHandle()
      attType := readReq.Type()


      //handles := make(GroupValueLst, 0, 100)
      for _,device := range this.devices {
        offset := uint16(device.handleOffset)
        if startHandle >= offset + 1 &&
           startHandle <= offset + uint16(len(device.handles)) {
          remoteReq := NewReadByTypeRequest(startHandle - offset,
                        endHandle - offset, attType)
          respBuf, _ := device.Transaction(remoteReq.msg)
          if respBuf[0] == ATT_OPCODE_ERROR {
            resp := NewError(ATT_OPCODE_READ_BY_TYPE_REQUEST, startHandle, 0x0A)
            req.device.Respond(resp.msg)
          } else {
            segLen := int(respBuf[1])
            for i := 2; i < len(respBuf); i += segLen {
              h := uint16(respBuf[i]) + uint16(respBuf[i + 1]) << 8
              h += offset
              respBuf[i] = byte(h & 0xff)
              respBuf[i + 1] = byte(h >> 8)
            }
            req.device.Respond(respBuf)
          }
          break
        }
      }
    case ATT_OPCODE_READ_REQUEST:
      fallthrough
    case ATT_OPCODE_READ_BLOB_REQUEST:
      fallthrough
    case ATT_OPCODE_WRITE_REQUEST:
      fallthrough
    case ATT_OPCODE_WRITE_COMMAND:
      fallthrough
    case ATT_OPCODE_SIGNED_WRITE_COMMAND:
      handleNum := uint16(pkt[1]) + uint16(pkt[2]) << 8
      var device *Device
      for _,d := range this.devices {
        if d.handleOffset + 1 < int(handleNum) &&
          len(d.handles) + d.handleOffset + 1 > int(handleNum) {
          device = d
          break
        }
      }
      if device == nil {
        resp := NewError(pkt[0], handleNum, 0x1)
        req.device.Respond(resp.msg)
        continue
      }
      remoteHandle := handleNum - uint16(device.handleOffset)
      if device.handles[remoteHandle] == nil {
        resp := NewError(pkt[0], handleNum, 0x1)
        req.device.Respond(resp.msg)
        continue
      }
      pkt[1] = byte(remoteHandle & 0xff)
      pkt[2] = byte(remoteHandle >> 8)
      go func() {
        if pkt[1] == ATT_OPCODE_WRITE_COMMAND || pkt[1] == ATT_OPCODE_SIGNED_WRITE_COMMAND {
          device.WriteCmd(pkt)
        } else {
          resp, err := device.Transaction(pkt)
          if err != nil {
            errResp := NewError(pkt[1], handleNum, 0x0E)
            req.device.Respond(errResp.msg)
          } else {
            req.device.Respond(resp)
          }
        }
      }()
    }
  }
}

func (this *Manager) connectTo(addr string) error {
  remoteAddr, err := Str2Ba(addr)
  if err != nil {
    return err
  }

  f, err := NewBLE(NewL2Sockaddr(4, remoteAddr, BDADDR_LE_RANDOM), addr)
  if err != nil {
    return err
  }

  device := NewDevice(addr, this.requestChan, f)
  this.devices = append(this.devices, device)

  return nil
}

func (this *Manager) start(idx int) error {
  if idx >= len(this.devices) || idx < 0 {
    return errors.New("No such device")
  }

  device := this.devices[idx]
  device.Start()
  device.StartClient()

  handles, err := DiscoverHandles(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  device.handleOffset = this.globalHandleOffset
  this.globalHandleOffset += len(handles)

  for _, handle := range handles {
    device.handles[handle.handle] = handle
  }

  groupVals, err := DiscoverServices(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  for _,v := range groupVals {
    device.handles[v.handle].cachedValue = v.value
    device.handles[v.handle].endGroup = v.endGroup
  }

  handleVals, err := DiscoverCharacteristics(device)
  if err != nil {
    device.fd.Close()
    return err
  }

  for _,v := range handleVals {
    device.handles[v.handle].cachedValue = v.value
  }

  return nil
}

func (this *Manager) disconnectFrom(idx int) error {
  if idx >= len(this.devices) || idx < 0 {
    return errors.New("No such device")
  }

  device := this.devices[idx]

  err := device.fd.Close()
  if err != nil {
    return err
  }

  this.devices = append(this.devices[0:idx], this.devices[idx + 1:]...)
  delete(this.associations, idx)
  for cl,lst := range this.associations {
    for i,v := range lst {
      if v == idx {
        this.associations[cl] = append(lst[0:i], lst[i + 1:]...)
        break
      }
    }
    if len(this.associations) == 0 {
      delete(this.associations, cl)
    }
  }
  return nil
}

func (this *Manager) serveTo(serverIdx, clientIdx int) error {
  if clientIdx >= len(this.devices) || serverIdx >= len(this.devices) {
    return errors.New("No such device")
  }

  if lst,ok := this.associations[clientIdx]; ok {
    this.associations[clientIdx] = append(lst, serverIdx)
  } else {
    this.associations[clientIdx] = []int {serverIdx}
  }

  return nil
}

