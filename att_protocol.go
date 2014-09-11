package main

import (
  "errors"
  "fmt"
  "io"
)

const (
  _ = iota
  ATT_OPCODE_ERROR uint8 = iota
  ATT_OPCODE_MTU_REQUEST
  ATT_OPCODE_MTU_RESPONSE
  ATT_OPCODE_FIND_INFO_REQUEST
  ATT_OPCODE_FIND_INFO_RESPONSE
  ATT_OPCODE_FIND_BY_TYPE_VALUE_REQUEST
  ATT_OPCODE_FIND_BY_TYPE_VALUE_RESPONSE
  ATT_OPCODE_READ_BY_TYPE_REQUEST
  ATT_OPCODE_READ_BY_TYPE_RESPONSE
  ATT_OPCODE_READ_REQUEST
  ATT_OPCODE_READ_RESPONSE
  ATT_OPCODE_READ_BLOB_REQUEST
  ATT_OPCODE_READ_BLOB_RESPONSE
  ATT_OPCODE_READ_MULTIPLE_REQUEST
  ATT_OPCODE_READ_MULTIPLE_RESPONSE
  ATT_OPCODE_READ_BY_GROUP_TYPE_REQUEST
  ATT_OPCODE_READ_BY_GROUP_TYPE_RESPONSE
  ATT_OPCODE_WRITE_REQUEST
  ATT_OPCODE_WRITE_RESPONSE

  ATT_OPCODE_WRITE_COMMAND = 0x52
  ATT_OPCODE_PREPARE_WRITE_REQUEST = 0x16
  ATT_OPCODE_PREPARE_WRITE_RESPONSE = 0x17
  ATT_OPCODE_EXECUTE_WRITE_REQUEST = 0x18
  ATT_OPCODE_EXECUTE_WRITE_RESPONSE = 0x19
  ATT_OPCODE_HANDLE_VALUE_NOTIFICATION = 0x1B
  ATT_OPCODE_HANDLE_VALUE_INDICATION = 0x1D
  ATT_OPCODE_HANDLE_VALUE_CONFIRMATION = 0x1E
  ATT_OPCODE_SIGNED_WRITE_COMMAND = 0xD2
)

type AttPDU interface {
  Msg()    []byte
}

type Error struct {
  msg    []byte
}

type HandleInfo struct {
  format uint8
  handle uint16
  cachedValue []byte
  uuid UUID
}

func NewError(msg []byte) (*Error, error) {
  if len(msg) == 5 {
    return &Error{msg}, nil
  } else {
    return nil, errors.New("Message must be 5 bytes")
  }
}

func (this *Error) Msg() []byte {
  return this.msg
}

func (this *Error) ReqOpcode() uint8 {
  return this.msg[1]
}

func (this *Error) Handle() uint16 {
  return uint16(this.msg[2]) + uint16(this.msg[3]) << 8
}

func (this *Error) ErrorCode() uint8 {
  return this.msg[4]
}

type FindInfoResponse struct {
  msg []byte
}

func ParseFindInfoResponse(msg []byte) (*FindInfoResponse, error) {
  if len(msg) >= 6 && (len(msg) - 2) % 4 == 0 {
    return &FindInfoResponse{msg}, nil
  } else {
    return nil, errors.New("Message is not the right length")
  }
}

func (this *FindInfoResponse) Msg() []byte {
  return this.msg
}

func (this *FindInfoResponse) Format() uint8 {
  return this.msg[1]
}

func (this *FindInfoResponse) InfoData() []*HandleInfo {
  format := this.Format()
  var step int
  if format == 1 {
    step = 4
  } else {
    step = 18
  }

  ihs := make([]*HandleInfo, 0)
  for i := 2; i < len(this.msg); i += step {
    buf := this.msg[i:i + step]

    handleNum := uint16(buf[0]) + uint16(buf[1]) << 16
    var uuid UUID
    uuid[0] = buf[2]
    uuid[1] = buf[3]
    if format == 2 {
      for j := 2; j < 16; j++ {
        uuid[j] = buf[j + 2]
      }
    }

    handle := &HandleInfo{}
    handle.format = format
    handle.handle = handleNum
    handle.uuid = uuid

    ihs = append(ihs, handle)
  }
  return ihs
}

type FindInfoByValueResponse struct {
  msg []byte
}

func NewFindInfoByValueResponse(msg []byte) (*FindInfoByValueResponse, error) {
  if len(msg) >= 7 {
    return &FindInfoByValueResponse{msg}, nil
  } else {
    return nil, errors.New("Message is not the right length")
  }
}

func (this *FindInfoByValueResponse) Msg() []byte {
  return this.msg
}

type ReadByGroupTypeResponse struct {
  msg []byte
}

func ParseReadByGroupTypeResponse(msg []byte) (*ReadByGroupTypeResponse, error) {
  if len(msg) >= 4 {
    return &ReadByGroupTypeResponse{msg}, nil
  } else {
    return nil, errors.New("Message is not the right length")
  }
}

type GroupValue struct {
  handle uint16
  endGroup uint16
  value []byte
}

func (this *ReadByGroupTypeResponse) Length() uint8 {
  return this.msg[1]
}

func (this *ReadByGroupTypeResponse) DataList() []*GroupValue {
  step := int(this.Length())
  length := step - 4

  vals := make([]*GroupValue, 0)
  for i := 2; i < len(this.msg); i += step {
    buf := this.msg[i:i + step]

    handle := uint16(buf[0]) + uint16(buf[1]) << 16
    endGroup := uint16(buf[2]) + uint16(buf[3]) << 16
    value := make([]byte, length)
    copy(value, buf[4:])

    groupVal := &GroupValue{}
    groupVal.handle = handle
    groupVal.endGroup = endGroup
    groupVal.value = value

    vals = append(vals, groupVal)
  }
  return vals
}

func DiscoverHandles(f io.ReadWriter) ([]*HandleInfo, error) {
  buf := make([]byte, 5)
  buf[0] = ATT_OPCODE_FIND_INFO_REQUEST

  var startHandle uint16 = 1
  var endHandle uint16   = 0xffff

  respBuf := make([]byte, 64)
  handles := make([]*HandleInfo, 0)
  for {
    // populate packet buffer
    buf[1] = byte(startHandle & 0xff)
    buf[2] = byte(startHandle >> 8)
    buf[3] = byte(endHandle & 0xff)
    buf[4] = byte(endHandle >> 8)

    _, err := f.Write(buf)
    if err != nil {
      return nil, err
    }

    n, err := f.Read(respBuf)
    if err != nil {
      return nil, err
    }
    resp := respBuf[0:n]

    if resp[0] == ATT_OPCODE_FIND_INFO_RESPONSE {
      fi, err := ParseFindInfoResponse(resp)
      if err != nil {
        return nil, err
      }
      handles = append(handles, fi.InfoData()...)

      startHandle = handles[len(handles) - 1].handle + 1
      continue
    }

    if resp[0] == ATT_OPCODE_ERROR  &&
       resp[1] == ATT_OPCODE_FIND_INFO_REQUEST && resp[4] == 0x0A {
        break
    } else {
      return nil, errors.New("Unexpected packet: " + string(resp))
    }
  }

  return handles, nil
}

func DiscoverServices(f io.ReadWriter) ([]*GroupValue, error) {
  buf := make([]byte, 7)
  buf[0] = ATT_OPCODE_READ_BY_GROUP_TYPE_REQUEST

  var startHandle uint16 = 1
  var endHandle uint16   = 0xffff

  buf[3] = byte(endHandle & 0xff)
  buf[4] = byte(endHandle >> 8)
  copy(buf[5:], []byte{0, 0x28}) // Primary Service UUID

  vals := make([]*GroupValue, 0)
  for {
    respBuf := make([]byte, 64)
    // populate packet buffer
    buf[1] = byte(startHandle & 0xff)
    buf[2] = byte(startHandle >> 8)

    _, err := f.Write(buf)
    if err != nil {
      return nil, err
    }

    n, err := f.Read(respBuf)
    if err != nil {
      return nil, err
    }
    resp := respBuf[0:n]

    if resp[0] == ATT_OPCODE_READ_BY_GROUP_TYPE_RESPONSE {
      fi, err := ParseReadByGroupTypeResponse(resp)
      if err != nil {
        return nil, err
      }
      vals = append(vals, fi.DataList()...)

      startHandle = vals[len(vals) - 1].endGroup + 1
      continue
    }

    if resp[0] == ATT_OPCODE_ERROR  &&
       resp[1] == ATT_OPCODE_READ_BY_GROUP_TYPE_REQUEST && resp[4] == 0x0A {
        break
    } else {
      str := fmt.Sprintf("%v", resp)
      return nil, errors.New("Unexpected packet: " + str)
    }
  }
  return vals, nil
}

