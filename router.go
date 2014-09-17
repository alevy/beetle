package main

import (
  "bytes"
  "sort"
)

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

