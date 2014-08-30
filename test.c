

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>
#include <bluetooth/l2cap.h>
#include <errno.h>
#include <uv.h>

#include "att_pdus.h"

void listen_for_le_events_start(int sock, struct hci_filter *old_filter) {
  struct hci_filter filter;
  socklen_t flen = sizeof(filter);

  // save socket state to restore for later
  if(getsockopt(sock, SOL_HCI, HCI_FILTER, old_filter, &flen)) {
    perror("getsockopt");
    exit(1);
  }

  // initialize filter
  hci_filter_clear(&filter);
  hci_filter_set_ptype(HCI_EVENT_PKT, &filter); // Packet type
  // All LE events are subevents of the LE meta event:
  hci_filter_set_event(EVT_LE_META_EVENT, &filter);

  if(setsockopt(sock, SOL_HCI, HCI_FILTER, &filter, sizeof(filter))) {
    perror("setsockopt hci filter");
    exit(1);
  }
}

void listen_for_le_events_cleanup(int sock, const struct hci_filter *old_filter) {
    // restore old filter
    if(setsockopt(sock, SOL_HCI, HCI_FILTER, old_filter, sizeof(old_filter))) {
      perror("setsockopt hci filter restore");
      exit(1);
    }
}

void get_advertized_device(int sock, bdaddr_t *dst_addr, uint8_t *type) {
  struct hci_filter old_filter;
  listen_for_le_events_start(sock, &old_filter);

  unsigned char buf[HCI_MAX_EVENT_SIZE], *ptr;
  int j;
  for(j = 0; j < 10; j++) {
    size_t len;
    evt_le_meta_event *meta_evt;
    if ((len = read(sock, buf, sizeof(buf))) < 0) {
      perror("read");
      exit(1);
    }

    ptr = buf + 1 + HCI_EVENT_HDR_SIZE;
    len -= 1 + HCI_EVENT_HDR_SIZE;

    meta_evt = (void*)ptr;

    if (meta_evt->subevent != 0x2) { // subevent 0x2 is Advertising Report Event
      break;
    }

    le_advertising_info *info = (le_advertising_info*)(meta_evt->data + 1);
    *type = info->bdaddr_type;
    printf("Addr type %d\n", *type);
    bacpy(dst_addr, &info->bdaddr);
  }

  listen_for_le_events_cleanup(sock, &old_filter);
}

int find_info(int s, uint16_t start, uint16_t end) {
  int size;
  struct find_info_req find_req_pdu;
  find_req_pdu.opcode = 0x4;
  find_req_pdu.start_handle = htole16(start);
  find_req_pdu.end_handle = htole16(end);
  size = write(s, (char*)(&find_req_pdu), sizeof(find_req_pdu));
  if (size < 0) {
    perror("write");
  }

  struct find_info_resp find_resp_pdu;
  size = read(s, (char*)(&find_resp_pdu), sizeof(find_resp_pdu));
  if (size < 0) {
    perror("read");
  }

  if (find_resp_pdu.opcode != 0x5) {
    return 0;
  }

  uint16_t handle = 0;
  if (find_resp_pdu.format == 1) {
    int num_handles = (size - 2) / 4;
    int i;
    for (i = 0; i < num_handles; ++i) {
      handle = le16toh(find_resp_pdu.handles16[i].handle);
      uint16_t uuid = le16toh(find_resp_pdu.handles16[i].uuid);
      printf("Handle 0x%x UUID %x\n", handle, uuid);
    }
  } else {
    printf("128 bit UUIDs\n");
  }
  return (int)(handle + 1);
}

void read_services(int fd) {
  struct read_by_16bit_type_pdu req;
  int size;
  int i;

  char req_buf[7];
  req_buf[0] = 0x08;
  req_buf[1] = 0x01;
  req_buf[2] = 0;
  req_buf[3] = 0xff;
  req_buf[4] = 0xff;
  req_buf[5] = 0x0;
  req_buf[6] = 0x28;
  req.opcode = 0x08;
  req.start_handle = htole16(1);
  req.start_handle = htole16(0xffff);
  req.att_type = htole16(0x2800);

  for (i = 0; i < sizeof(req); i ++) {
    printf("0x%1x ", req_buf[i]);
  }


  size = write(fd, req_buf, sizeof(req));
  if (size < 0) {
    perror("write");
  }

  char buf[48];
  read(fd, buf, 48);
  for (i = 0; i < 48; i ++) {
    if (buf[i] == 0x0a || 1) {
      printf("0x%1x ", (unsigned)buf[i]);
    }
  }
  printf("\n");

}

int main(int argc, char **argv)
{
    int dev_id, sock, len, flags;

    dev_id = hci_get_route(NULL);
    sock = hci_open_dev( dev_id );
    if (dev_id < 0 || sock < 0) {
        perror("opening socket");
        exit(1);
    }

    uint16_t scanInterval = htobs(0x10);
    uint16_t scanWindow = htobs(0x10);


    // LE scan parameters: Core spec 4.0, Part E, Chapter 7.8.10
    int err = hci_le_set_scan_parameters(
                sock,
                0x0, //passive scan
                scanInterval,
                scanWindow,
                0x0, // use own public address
                0x0, // accept all address (don't use whitelist)
                1000); // timeout in ms passed to UNIX poll
    if (err) {
      perror("set scan parameters");
      exit(1);
    }

    if (hci_le_set_scan_enable(
                sock,
                0x1, // enable
                0x1, // filter duplicates
                1000)) { // timeout in ms passed to UNIX poll
      perror("start scanning");
      exit(1);
    }

    /* grab scanned devices */
    bdaddr_t dst_addr;
    uint8_t dst_addr_type;
    get_advertized_device(sock, &dst_addr, &dst_addr_type);
    {
      char addr_n[18];
      ba2str(&dst_addr, addr_n);
      printf("Address: %s\n", addr_n);
    }

    err = hci_le_set_scan_enable(
                sock,
                0x0, // disable
                0x1, // filter duplicates
                1000); // timeout in ms passed to UNIX poll
    if (err) {
      perror("stop scanning");
      exit(1);
    }
    close(sock);

    /* Connect to ATT over L2CAP */

    int s = socket(AF_BLUETOOTH, SOCK_SEQPACKET, BTPROTO_L2CAP);
    if (s < 0) {
      perror("socket");
      exit(1);
    }

    struct sockaddr_l2 bind_addr = { 0 };

    bind_addr.l2_family = AF_BLUETOOTH;
    bind_addr.l2_cid = htobs(4); // ATT CID
    bacpy(&bind_addr.l2_bdaddr, BDADDR_ANY);
    bind_addr.l2_bdaddr_type = BDADDR_LE_PUBLIC;

    err = bind(s, (struct sockaddr*)&bind_addr, sizeof(bind_addr));
    if (err) {
      perror("L2CAP bind");
      close(s);
      exit(1);
    }

    {
      int flags;
      socklen_t len = sizeof(flags);
      if (getsockopt(s, SOL_SOCKET, L2CAP_LM, &flags, &len)) {
        perror("L2CAP setup");
        close(s);
        exit(1);
      }
      printf("%d\n", flags);
      flags |= L2CAP_LM_MASTER;
      printf("%d\n", flags);
      if (setsockopt(s, SOL_L2CAP, L2CAP_LM, &flags, len)) {
        perror("L2CAP setup");
        close(s);
        exit(1);
      }

      /*struct bt_security sec = { 0 };
      sec.level = BT_SECURITY_LOW;
      if (setsockopt(s, SOL_BLUETOOTH, BT_SECURITY, &sec, sizeof(sec))) {
        perror("set security");
        close(s);
        exit(1);
      }*/

      int opt = L2CAP_LM_AUTH;
      if (setsockopt(s, SOL_L2CAP, L2CAP_LM, &opt, sizeof(opt))) {
        perror("set l2cap security");
        close(s);
        exit(1);
      }
    }

    struct sockaddr_l2 conn_addr = { 0 };
    conn_addr.l2_family = AF_BLUETOOTH;
    conn_addr.l2_cid = htobs(4); // ATT CID
    int i;
    for (i = 0; i < 6; i++) {
      printf("%2.2X:", dst_addr.b[i]);
    }
    printf("\n");
    bacpy(&conn_addr.l2_bdaddr, &dst_addr);
    conn_addr.l2_bdaddr_type = BDADDR_LE_RANDOM;

    err = connect(s, (struct sockaddr*)&conn_addr, sizeof(conn_addr));
    if (err) {
      if (!(errno & (EINPROGRESS | EAGAIN))) {
        perror("L2CAP connect");
        close(s);
        exit(1);
      }
    }

    uint16_t start = 1;
    do {
      start = find_info(s, start, 0xffff);
    } while (start > 1);

    read_services(s);

    close(s);

    return 0;
}

static uv_loop_t *loop;
static char console_buf[1024];

void test_alloc(uv_handle_t *h, size_t ssize, uv_buf_t* buf) {
  buf->base = console_buf;
  buf->len = 1024;
}

void on_read_line(uv_stream_t* stdin_pipe, ssize_t nread, const uv_buf_t *buf) {
  if (nread == UV_EOF) {
    uv_close((uv_handle_t*)stdin_pipe, NULL);
  } else {
    printf("> ");
    fflush(stdout);
  }
  if (buf->base) {
    buf->base[0] = 0;
  }
}

