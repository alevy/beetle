package main

import (
  "errors"
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

type MTURequest struct {
  msg []byte
}

func NewMTURequest(msg []byte) (*MTURequest, error) {
  if len(msg) == 3 {
    return &MTURequest{msg}, nil
  } else {
    return nil, errors.New("Message must be 5 bytes")
  }
}

func (this *MTURequest) Msg() []byte {
  return this.msg
}

type MTUResponse struct {
  msg []byte
}

func NewMTUResponse(msg []byte) (*MTUResponse, error) {
  if len(msg) == 3 {
    return &MTUResponse{msg}, nil
  } else {
    return nil, errors.New("Message must be 5 bytes")
  }
}

func (this *MTUResponse) Msg() []byte {
  return this.msg
}

type FindInfoRequest struct {
  msg []byte
}

func NewFindInfoRequest(msg []byte) (*FindInfoRequest, error) {
  if len(msg) == 5 {
    return &FindInfoRequest{msg}, nil
  } else {
    return nil, errors.New("Message must be 5 bytes")
  }
}

func (this *FindInfoRequest) Msg() []byte {
  return this.msg
}

type FindInfoResponse struct {
  msg []byte
}

func NewFindInfoResponse(msg []byte) (*FindInfoResponse, error) {
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

func (this *FindInfoResponse) InfoData() []HandleInfo {
  format := this.Format()
  var num int
  if format == 1 {
    num = (len(this.msg) - 2) / 4
  } else {
    num = (len(this.msg) - 2) / 18
  }

  ihs := make([]HandleInfo, num)
  for i := 0; i < num; i++ {
    var buf []byte
    if format == 1 {
      buf = this.msg[i * 4 + 2:4]
    } else {
      buf = this.msg[i * 18 + 2:18]
    }
    handleNum := uint16(buf[0]) + uint16(buf[1]) << 16
    var uuid UUID
    uuid[0] = buf[2]
    uuid[1] = buf[3]
    if format == 2 {
      for j := 2; j < 16; j++ {
        uuid[j] = buf[j + 2]
      }
    }

    var handle HandleInfo
    handle.format = format
    handle.handle = handleNum
    handle.uuid = uuid

    ihs[i] = handle
  }
  return ihs
}

type FindInfoByValueRequest struct {
  msg []byte
}

func NewFindInfoByValueRequest(msg []byte) (*FindInfoByValueRequest, error) {
  if len(msg) >= 7 {
    return &FindInfoByValueRequest{msg}, nil
  } else {
    return nil, errors.New("Message is not the right length")
  }
}

func (this *FindInfoByValueRequest) Msg() []byte {
  return this.msg
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

func ParseMessage(msg []byte) (pdu AttPDU, err error) {
  if len(msg) == 0 {
    err = errors.New("Message cannot be empty")
    return
  }

  switch msg[0] {
  case ATT_OPCODE_ERROR:
    pdu, err = NewError(msg)
  case ATT_OPCODE_MTU_REQUEST:
    pdu, err = NewMTURequest(msg)
  case ATT_OPCODE_MTU_RESPONSE:
    pdu, err = NewMTUResponse(msg)
  case ATT_OPCODE_FIND_INFO_REQUEST:
    pdu, err = NewFindInfoRequest(msg)
  case ATT_OPCODE_FIND_INFO_RESPONSE:
    pdu, err = NewFindInfoResponse(msg)
  case ATT_OPCODE_FIND_BY_TYPE_VALUE_REQUEST:
    pdu, err = NewFindInfoByValueRequest(msg)
  case ATT_OPCODE_FIND_BY_TYPE_VALUE_RESPONSE:
    pdu, err = NewFindInfoByValueResponse(msg)
  case ATT_OPCODE_READ_BY_TYPE_REQUEST:
  case ATT_OPCODE_READ_BY_TYPE_RESPONSE:
  case ATT_OPCODE_READ_REQUEST:
  case ATT_OPCODE_READ_RESPONSE:
  case ATT_OPCODE_READ_BLOB_REQUEST:
  case ATT_OPCODE_READ_BLOB_RESPONSE:
  case ATT_OPCODE_READ_MULTIPLE_REQUEST:
  case ATT_OPCODE_READ_MULTIPLE_RESPONSE:
  case ATT_OPCODE_READ_BY_GROUP_TYPE_REQUEST:
  case ATT_OPCODE_READ_BY_GROUP_TYPE_RESPONSE:
  case ATT_OPCODE_WRITE_REQUEST:
  case ATT_OPCODE_WRITE_RESPONSE:
  case ATT_OPCODE_WRITE_COMMAND:
  case ATT_OPCODE_PREPARE_WRITE_REQUEST:
  case ATT_OPCODE_PREPARE_WRITE_RESPONSE:
  case ATT_OPCODE_EXECUTE_WRITE_REQUEST:
  case ATT_OPCODE_EXECUTE_WRITE_RESPONSE:
  case ATT_OPCODE_HANDLE_VALUE_NOTIFICATION:
  case ATT_OPCODE_HANDLE_VALUE_INDICATION:
  case ATT_OPCODE_HANDLE_VALUE_CONFIRMATION:
  case ATT_OPCODE_SIGNED_WRITE_COMMAND:
  default:
    err = errors.New("Bad message opcode")
  }

  return
}
