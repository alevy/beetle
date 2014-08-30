
struct find_info_req {
  uint8_t opcode;
  uint16_t start_handle;
  uint16_t end_handle;
} __attribute__ ((packed));

struct h16 {
  uint16_t handle;
  uint16_t uuid;
};

struct h128 {
  uint16_t handle;
  uint64_t uuid[2];
};

struct find_info_resp {
  uint8_t   opcode;
  uint8_t   format;
  union {
    struct h16 handles16[(L2CAP_DEFAULT_MTU - 2) / sizeof(struct h16)];
    struct h128 handles128[(L2CAP_DEFAULT_MTU - 2) / sizeof(struct h128)];
  };
} __attribute__ ((packed));

struct read_by_16bit_type_pdu {
  uint8_t opcode;
  uint16_t start_handle;
  uint16_t end_handle;
  uint16_t att_type;
} __attribute__ ((packed));

struct read_by_64bit_type_pdu {
  uint8_t opcode;
  uint16_t start_handle;
  uint16_t end_handle;
  uint64_t att_type[2];
} __attribute__ ((packed));

