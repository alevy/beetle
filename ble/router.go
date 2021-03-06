package ble

import (
  "bytes"
  "time"
  "sort"
)

func (this *Manager) RouteConnUpdate(req Request) {
  interval := uint16(req.msg[1]) + uint16(req.msg[2]) << 8
  for _,device := range(this.Devices) {
    if device != req.device {
      this.ConnUpdate(device, interval)
    }
  }
  msg := make([]byte,1)
  msg[0] = 0xff
  req.device.Respond(msg)
}

func (this *Manager) RouteFindInfo(req Request) {
  findReq, err := ParseFindInfoRequest(req.msg)
  if err != nil {
    resp := NewError(ATT_OPCODE_FIND_INFO_REQUEST, 0, 4)
    req.device.Respond(resp.msg)
    return
  } else {
    startHandle := findReq.StartHandle()
    endHandle := findReq.EndHandle()

    handles := make(HandleUUIDLst, 0, 10)
    for _, device := range this.Devices {
      offset := uint16(device.handleOffset)

      if device == req.device || device.handleOffset < 0 ||
        offset + uint16(len(device.handles)) < startHandle {
        continue
      }
      if device.handleOffset > int(endHandle) {
        break
      }

      for _, handle := range device.handles {
        if handle.handle + offset >= startHandle &&
            handle.handle + offset <= endHandle {
          h := HandleUUID{handle.handle + offset, handle.uuid}
          h.handle += offset
          handles = append(handles, h)
        }
      }
    }

    sort.Sort(handles)

    if len(handles) > 0 {
      resp := NewFindInfoResponse(handles)
      req.device.Respond(resp.msg)
    } else {
      resp := NewError(ATT_OPCODE_FIND_INFO_REQUEST, startHandle, 0x0A)
      req.device.Respond(resp.msg)
    }
  }
}


func (this *Manager) RouteFindByTypeValue(req Request) {
  findReq, err := ParseFindByTypeValueRequest(req.msg)
  if err != nil {
    resp := NewError(ATT_OPCODE_FIND_BY_TYPE_VALUE_REQUEST, 0, 4)
    req.device.Respond(resp.msg)
    return
  } else {
    startHandle := findReq.StartHandle()
    endHandle := findReq.EndHandle()
    attType := findReq.Type()
    attVal := findReq.Value()

    handles := make(GroupValueLst, 0, 10)
    for _,device := range this.Devices {
      offset := uint16(device.handleOffset)

      if device == req.device || device.handleOffset < 0 ||
        offset + uint16(len(device.handles)) < startHandle {
        continue
      }
      if offset > endHandle {
        break
      }

      for _, handle := range device.handles {
        if handle.handle + offset >= startHandle &&
            handle.handle + offset <= endHandle &&
            handle.uuid == attType && bytes.Equal(handle.cachedValue, attVal) {
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
  }
}

func (this *Manager) RouteReadByType(req Request) {
  readReq, err := ParseReadByTypeRequest(req.msg)
  if err != nil {
    resp := NewError(ATT_OPCODE_READ_BY_TYPE_REQUEST, 0, 4)
    req.device.Respond(resp.msg)
    return
  }

  startHandle := readReq.StartHandle()
  endHandle := readReq.EndHandle()
  attType := readReq.Type()


  for _,device := range this.Devices {
    if (device == req.device) {
      continue
    }
    offset := uint16(device.handleOffset)
    if startHandle >= offset + 1 &&
       startHandle <= offset + uint16(len(device.handles)) {
      remoteReq := NewReadByTypeRequest(startHandle - offset,
                    endHandle - offset, attType)
      device.Transaction(remoteReq.msg, func(respBuf []byte, err error) {
        if err != nil {
          resp := NewError(ATT_OPCODE_READ_BY_TYPE_REQUEST, 0, 4)
          req.device.Respond(resp.msg)
          return
        }
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

            if attType == GATT_CHARACTERISTIC_UUID {
              j := i + 3
              value_handle := uint16(respBuf[j]) + uint16(respBuf[j + 1]) << 8
              value_handle += offset
              respBuf[j] = byte(value_handle & 0xff)
              respBuf[j + 1] = byte(value_handle >> 8)
            }
          }
          req.device.Respond(respBuf)
        }
      })
      return
    }
  }
  req.device.Respond(NewError(ATT_OPCODE_READ_BY_TYPE_REQUEST, 0, 0x0A).msg)
}

func (this *Manager) RunRouter() {
  //interval := uint16(6)
  for req := range this.requestChan {
    pkt := req.msg
    switch(pkt[0]) {
    case ATT_OPCODE_CONN_UPDATE:
      this.RouteConnUpdate(req)
    case ATT_OPCODE_FIND_INFO_REQUEST:
      this.RouteFindInfo(req)
    case ATT_OPCODE_FIND_BY_TYPE_VALUE_REQUEST:
      this.RouteFindByTypeValue(req)
    case ATT_OPCODE_READ_BY_TYPE_REQUEST:
      go this.RouteReadByType(req)

    case ATT_OPCODE_HANDLE_VALUE_NOTIFICATION:
      handleNum := uint16(pkt[1]) + uint16(pkt[2]) << 8
      device := req.device

      proxyHandle := device.handles[handleNum]

      if proxyHandle == nil {
        continue
      }

      remoteHandle := handleNum + uint16(device.handleOffset)
      pkt[1] = byte(remoteHandle & 0xff)
      pkt[2] = byte(remoteHandle >> 8)

      for dev,_ := range proxyHandle.subscribers {
        dev.WriteCmd(pkt)
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
      for _,d := range this.Devices {
        if d.handleOffset < int(handleNum) &&
          d.highestHandle >= int(handleNum) {
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
      proxyHandle := device.handles[remoteHandle]

      if proxyHandle == nil {
        resp := NewError(pkt[0], handleNum, 0x1)
        req.device.Respond(resp.msg)
        continue
      }

      if pkt[0] == ATT_OPCODE_WRITE_REQUEST &&
         proxyHandle.uuid == GATT_CLIENT_CONFIGURATION_UUID {
        proxyCharHandle := device.handles[proxyHandle.charHandle]
        proxyCharHandle = device.handles[proxyCharHandle.charHandle]
        if pkt[3] == 0 {
          delete(proxyCharHandle.subscribers, req.device)
        } else {
          proxyCharHandle.subscribers[req.device] = true
        }
        numSubscribers := len(proxyCharHandle.subscribers)
        if numSubscribers == 0 && pkt[3] == 0 ||
           numSubscribers == 1 && pkt[3] == 1 {
          // First subscribe or last unsubscribe case
          // Translate packet handle with sutracted device offset
          pkt[1] = byte(remoteHandle & 0xff)
          pkt[2] = byte(remoteHandle >> 8)

          device.Transaction(pkt, func(resp []byte, err error) {
            if err != nil {
              errResp := NewError(pkt[1], handleNum, 0x0E)
              req.device.Respond(errResp.msg)
            } else {
              req.device.Respond(resp)
            }
          })
        } else {
          req.device.Respond([]byte{ATT_OPCODE_WRITE_RESPONSE})
        }
      } else if false && pkt[0] == ATT_OPCODE_READ_REQUEST && proxyHandle.cachedValue != nil &&
          (/*&& time.Since(proxyHandle.cachedTime) <=
             time.Duration(interval) * time.Millisecond*/
          !proxyHandle.cachedMap[req.device] || proxyHandle.cachedInfinite)  {
        proxyHandle.cachedMap[req.device] = true
        resp := make([]byte, 1 + len(proxyHandle.cachedValue))
        resp[0] = ATT_OPCODE_READ_RESPONSE
        copy(resp[1:], proxyHandle.cachedValue)
        req.device.Respond(resp)
      } else {
        pkt[1] = byte(remoteHandle & 0xff)
        pkt[2] = byte(remoteHandle >> 8)
        if pkt[0] == ATT_OPCODE_WRITE_COMMAND || pkt[0] == ATT_OPCODE_SIGNED_WRITE_COMMAND {
          device.WriteCmd(pkt)
        } else {
          device.Transaction(pkt, func(resp []byte, err error) {
            if err != nil {
              errResp := NewError(pkt[1], handleNum, 0x0E)
              req.device.Respond(errResp.msg)
            } else {
              if resp[0] == ATT_OPCODE_READ_RESPONSE {
                proxyHandle.cachedMap = make(map[*Device]bool)
                proxyHandle.cachedMap[req.device] = true
                proxyHandle.cachedValue = resp[1:]
                proxyHandle.cachedTime = time.Now()
              }
              req.device.Respond(resp)
            }
          })
        }
      }
    }
  }
}

