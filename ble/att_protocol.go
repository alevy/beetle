package ble

import (
  "errors"
  "fmt"
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

type UUID [16]uint8

var BLUETOOTH_BASE_UUID [12]byte =
  [12]byte{0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5F, 0x9B, 0x34, 0xFB}
var GATT_PRIMARY_SERVICE_UUID UUID =
  [16]byte{0, 0, 0x0, 0x28, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5F, 0x9B, 0x34, 0xFB}
var GATT_CHARACTERISTIC_UUID UUID =
  [16]byte{0, 0, 0x3, 0x28, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5F, 0x9B, 0x34, 0xFB}

var GATT_CLIENT_CONFIGURATION_UUID UUID =
  [16]byte{0, 0, 0x2, 0x29, 0x00, 0x00, 0x10, 0x00, 0x80, 0x00, 0x00, 0x80, 0x5F, 0x9B, 0x34, 0xFB}

type HandleInfo struct {
  format uint8
  handle uint16
  endGroup uint16
  cachedValue []byte
  uuid UUID
}

type Error struct {
  msg    []byte
}

func NewError(reqOpcode uint8, handle uint16, errCode uint8) (*Error) {
  return &Error{[]byte{ATT_OPCODE_ERROR,
                       reqOpcode,
                       byte(handle & 0xff), byte(handle >> 8),
                       errCode}}
}

func ParseError(msg []byte) (*Error, error) {
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

type FindInfoRequest struct {
  msg []byte
}

func ParseFindInfoRequest(msg []byte) (*FindInfoRequest, error) {
  if len(msg) != 5 {
    return nil, errors.New("Message must be 5 octects")
  }

  return &FindInfoRequest{msg}, nil
}

func (this *FindInfoRequest) StartHandle() uint16 {
  return uint16(this.msg[1]) + uint16(this.msg[2]) << 8
}

func (this *FindInfoRequest) EndHandle() uint16 {
  return uint16(this.msg[3]) + uint16(this.msg[4]) << 8
}

type HandleUUID struct {
  handle uint16
  uuid UUID
}

type HandleUUIDLst []HandleUUID

func (this HandleUUIDLst) Len() int {
  return len(this)
}

func (this HandleUUIDLst) Less(i, j int) bool {
  return this[i].handle < this[j].handle
}

func (this HandleUUIDLst) Swap(i, j int) {
  tmp := this[j]
  this[j] = this[i]
  this[i] = tmp
}

type FindInfoResponse struct {
  msg []byte
}

func NewFindInfoResponse(handles []HandleUUID) (*FindInfoResponse) {
  msg := make([]byte, 24)
  msg[0] = ATT_OPCODE_FIND_INFO_RESPONSE
  msg[1] = 1
  i := 2
  for _, handle := range handles {
    if i > len(msg) - 4 {
      break
    }

    msg[i] = byte(handle.handle & 0xff)
    i++
    msg[i] = byte(handle.handle >> 8)
    i++
    msg[i] = handle.uuid[2]
    i++
    msg[i] = handle.uuid[3]
    i++
  }

  return &FindInfoResponse{msg[0:i]}
}

func ParseFindInfoResponse(msg []byte) (*FindInfoResponse, error) {
  if len(msg) >= 6 {
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
    if format == 2 {
      for j := 0; j < 16; j++ {
        uuid[j] = buf[j]
      }
    } else {
      uuid[2] = buf[2]
      uuid[3] = buf[3]

      for j := 4; j < 16; j++ {
        uuid[j] = BLUETOOTH_BASE_UUID[j - 4]
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

type FindByTypeValueRequest struct {
  msg []byte
}

func ParseFindByTypeValueRequest(msg []byte) (*FindByTypeValueRequest, error) {
  if len(msg) >= 7 {
    return &FindByTypeValueRequest{msg}, nil
  } else {
    return nil, errors.New("Message is not the right length")
  }
}

func (this *FindByTypeValueRequest) StartHandle() uint16 {
  return uint16(this.msg[1]) | uint16(this.msg[2]) << 8
}

func (this *FindByTypeValueRequest) EndHandle() uint16 {
  return uint16(this.msg[3]) | uint16(this.msg[4]) << 8
}

func (this *FindByTypeValueRequest) Type() UUID {
  var uuid UUID
  uuid[2] = this.msg[5]
  uuid[3] = this.msg[6]

  for j := 4; j < 16; j++ {
    uuid[j] = BLUETOOTH_BASE_UUID[j - 4]
  }
  return uuid
}

func (this *FindByTypeValueRequest) Value() []byte {
  return this.msg[7:]
}

type FindByTypeValueResponse struct {
  msg []byte
}

func NewFindByTypeValueResponse(vals []*GroupValue) (*FindByTypeValueResponse) {
  msg := make([]byte, 24)
  msg[0] = ATT_OPCODE_FIND_BY_TYPE_VALUE_RESPONSE

  i := 1
  cutShort := false
  for _, val := range vals {
    if i > 20 {
      cutShort = true
      break
    }
    msg[i] = byte(val.handle & 0xff)
    i++
    msg[i] = byte(val.handle >> 8)
    i++
    msg[i] = byte(val.endGroup & 0xff)
    i++
    msg[i] = byte(val.endGroup >> 8)
    i++
  }

  if !cutShort {
    msg[i - 1] = 0xff
    msg[i - 2] = 0xff
  }

  return &FindByTypeValueResponse{msg[0:i]}
}

type ReadByTypeRequest struct {
  msg []byte
}

func NewReadByTypeRequest(startHandle, endHandle uint16, attType UUID) (*ReadByTypeRequest){
  msg := make([]byte, 7)
  msg[0] = ATT_OPCODE_READ_BY_TYPE_REQUEST
  msg[1] = byte(startHandle & 0xff)
  msg[2] = byte(startHandle >> 8)
  msg[3] = byte(endHandle & 0xff)
  msg[4] = byte(endHandle >> 8)
  msg[5] = attType[2]
  msg[6] = attType[3]
  return &ReadByTypeRequest{msg}
}

func ParseReadByTypeRequest(msg []byte) (*ReadByTypeRequest, error) {
  if len(msg) == 7 || len(msg) == 21 {
    return &ReadByTypeRequest{msg}, nil
  } else {
    return nil, errors.New("Message is not the right length")
  }
}

func (this *ReadByTypeRequest) StartHandle() uint16 {
  return uint16(this.msg[1]) | uint16(this.msg[2]) << 8
}

func (this *ReadByTypeRequest) EndHandle() uint16 {
  return uint16(this.msg[3]) | uint16(this.msg[4]) << 8
}

func (this *ReadByTypeRequest) Type() UUID {
  var uuid UUID
  if len(this.msg) == 7 {
    uuid[2] = this.msg[5]
    uuid[3] = this.msg[6]

    for j := 4; j < 16; j++ {
      uuid[j] = BLUETOOTH_BASE_UUID[j - 4]
    }
  } else {
    for i := 0; i < 16; i++ {
      uuid[i] = this.msg[5 + i]
    }
  }
  return uuid
}

type ReadByTypeResponse struct {
  msg []byte
}

func NewReadByTypeResponse(vals []*GroupValue) (*ReadByTypeResponse) {
  msg := make([]byte, 24)
  msg[0] = ATT_OPCODE_READ_BY_TYPE_RESPONSE

  baseLen := len(vals[0].value)
  msg[1] = byte(baseLen) + 2
  i := 2
  for _, val := range vals {
    if len(val.value) != baseLen {
      break
    }
    if i > len(msg) - 2 - len(val.value) {
      break
    }
    msg[i] = byte(val.handle & 0xff)
    i++
    msg[i] = byte(val.handle >> 8)
    i++
    copy(msg[i:], val.value)
    i += len(val.value)
  }

  return &ReadByTypeResponse{msg[0:i]}
}

func ParseReadByTypeResponse(msg []byte) (*ReadByTypeResponse, error) {
  if len(msg) >= 4 {
    return &ReadByTypeResponse{msg}, nil
  } else {
    return nil, errors.New("Message is not the right length")
  }
}


type GroupValue struct {
  handle uint16
  endGroup uint16
  value []byte
}

type GroupValueLst []*GroupValue
func (this GroupValueLst) Len() int {
  return len(this)
}

func (this GroupValueLst) Less(i, j int) bool {
  return this[i].handle < this[j].handle
}

func (this GroupValueLst) Swap(i, j int) {
  tmp := this[i]
  this[i] = this[j]
  this[j] = tmp
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

type HandleValue struct {
  handle uint16
  value []byte
}

func (this *ReadByTypeResponse) Length() uint8 {
  return this.msg[1]
}

func (this *ReadByTypeResponse) DataList() []*HandleValue {
  step := int(this.Length())
  length := step - 2

  vals := make([]*HandleValue, 0)
  for i := 2; i < len(this.msg); i += step {
    buf := this.msg[i:i + step]

    handle := uint16(buf[0]) + uint16(buf[1]) << 16
    value := make([]byte, length)
    copy(value, buf[2:])

    groupVal := &HandleValue{}
    groupVal.handle = handle
    groupVal.value = value

    vals = append(vals, groupVal)
  }
  return vals
}

func DiscoverServices(f *Device) ([]*GroupValue, error) {
  var startHandle uint16 = 1
  var endHandle uint16   = 0xffff

  vals := make([]*GroupValue, 0, 4)
  for {
    buf := make([]byte, 7)
    // populate packet buffer
    buf[0] = ATT_OPCODE_READ_BY_GROUP_TYPE_REQUEST
    buf[1] = byte(startHandle & 0xff)
    buf[2] = byte(startHandle >> 8)
    buf[3] = byte(endHandle & 0xff)
    buf[4] = byte(endHandle >> 8)
    copy(buf[5:], []byte{0, 0x28}) // Primary Service UUID

    r := make(chan Response)
    f.transactChan<-Transaction{buf, r}
    respS := <-r
    err := respS.err
    resp := respS.value

    if err != nil {
      return nil, err
    }

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

func DiscoverCharacteristics(f *Device, startHandle uint16,
        endHandle uint16) ([]*HandleValue, error) {
  vals := make([]*HandleValue, 0, 4)
  for {
    buf := make([]byte, 7)
    // populate packet buffer
    buf[0] = ATT_OPCODE_READ_BY_TYPE_REQUEST
    buf[1] = byte(startHandle & 0xff)
    buf[2] = byte(startHandle >> 8)


    buf[3] = byte(endHandle & 0xff)
    buf[4] = byte(endHandle >> 8)
    copy(buf[5:], []byte{3, 0x28}) // Characteristic Decleration

    r := make(chan Response)
    f.transactChan<-Transaction{buf, r}
    respS := <-r
    err := respS.err
    resp := respS.value

    if err != nil {
      return nil, err
    }

    if resp[0] == ATT_OPCODE_READ_BY_TYPE_RESPONSE {
      fi, err := ParseReadByTypeResponse(resp)
      if err != nil {
        return nil, err
      }
      vals = append(vals, fi.DataList()...)

      startHandle = vals[len(vals) - 1].handle + 1
      continue
    }

    if resp[0] == ATT_OPCODE_ERROR  &&
       resp[1] == ATT_OPCODE_READ_BY_TYPE_REQUEST && resp[4] == 0x0A {
        break
    } else {
      str := fmt.Sprintf("%v", resp)
      return nil, errors.New("Unexpected packet: " + str)
    }
  }
  return vals, nil
}

func DiscoverHandles(f *Device, startHandle uint16,
        endHandle uint16) ([]*HandleInfo, error) {
  handles := make([]*HandleInfo, 0)
  for {
    buf := make([]byte, 5)
    buf[0] = ATT_OPCODE_FIND_INFO_REQUEST

    // populate packet buffer
    buf[1] = byte(startHandle & 0xff)
    buf[2] = byte(startHandle >> 8)
    buf[3] = byte(endHandle & 0xff)
    buf[4] = byte(endHandle >> 8)

    r := make(chan Response)
    f.transactChan<-Transaction{buf, r}
    respS := <-r
    err := respS.err
    resp := respS.value

    if err != nil {
      return nil, err
    }

    if resp[0] == ATT_OPCODE_FIND_INFO_RESPONSE {
      fi, err := ParseFindInfoResponse(resp)
      if err != nil {
        return nil, err
      }
      handles = append(handles, fi.InfoData()...)

      startHandle = handles[len(handles) - 1].handle + 1
      if startHandle >= endHandle {
        break
      } else {
        continue
      }
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

