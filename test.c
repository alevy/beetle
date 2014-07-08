

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>
#include <bluetooth/l2cap.h>
#include <errno.h>

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
  for(j = 0; j < 1; j++) {
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
    bacpy(dst_addr, &info->bdaddr);
  }

  listen_for_le_events_cleanup(sock, &old_filter);
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

    /* Create connection */

    /*uint16_t minConnInterval = htobs(0x0006);
    uint16_t maxConnInterval = htobs(0x0006);
    uint16_t slaveLatency = htobs(0x0000);
    uint16_t supervisionTimeout = htobs(0x0C80);
    uint16_t minCELength = htobs(0x0001);
    uint16_t maxCELength = htobs(0x0001);
    uint16_t handle;

    err = hci_le_create_conn(
            sock,
            scanInterval,
            scanWindow,
            0x0, // Don't use whitelist
            dst_addr_type,
            dst_addr,
            0x0, // Public self address
            minConnInterval, // Minimum connection interval
            maxConnInterval, // Maximum connection interval
            slaveLatency,
            supervisionTimeout, minCELength, maxCELength, &handle, 2500);
    if (err) {
      perror("connect");
      exit(1);
    }*/
    close(sock);

    /* Connect to ATT over L2CAP */

    int s = socket(PF_BLUETOOTH, SOCK_SEQPACKET, BTPROTO_L2CAP);
    if (s < 0) {
      perror("socket");
      goto finish;
    }

    struct sockaddr_l2 bind_addr = { 0 };

    bind_addr.l2_family = AF_BLUETOOTH;
    bind_addr.l2_cid = htobs(4); // ATT CID
    bacpy(&bind_addr.l2_bdaddr, BDADDR_ANY);
    bind_addr.l2_bdaddr_type = BDADDR_LE_PUBLIC;

    err = bind(s, (struct sockaddr*)&bind_addr, sizeof(bind_addr));
    if (err) {
      perror("L2CAP bind");
      goto finish;
    }

    {
      int flags;
      socklen_t len = sizeof(flags);
      if (getsockopt(s, SOL_SOCKET, L2CAP_LM, &flags, &len)) {
        perror("L2CAP setup");
        goto finish;
      }
      flags |= L2CAP_LM_MASTER;
      if (setsockopt(s, SOL_L2CAP, L2CAP_LM, &flags, len)) {
        perror("L2CAP setup");
        goto finish;
      }

      struct bt_security sec = { 0 };
      sec.level = BT_SECURITY_LOW;
      if (setsockopt(s, SOL_BLUETOOTH, BT_SECURITY, &sec, sizeof(sec))) {
        perror("set security");
        goto finish;
      }

      int opt = L2CAP_LM_AUTH;
      if (setsockopt(s, SOL_L2CAP, L2CAP_LM, &opt, sizeof(opt))) {
        perror("set l2cap security");
        goto finish;
      }
    }

    struct sockaddr_l2 conn_addr = { 0 };
    conn_addr.l2_family = AF_BLUETOOTH;
    conn_addr.l2_cid = htobs(4); // ATT CID
    bacpy(&conn_addr.l2_bdaddr, &dst_addr);
    conn_addr.l2_bdaddr_type = BDADDR_LE_RANDOM;
    printf("Type %d\n", dst_addr_type & BDADDR_LE_RANDOM);

    err = connect(s, (struct sockaddr*)&conn_addr, sizeof(conn_addr));
    if (err) {
      if (!(errno & (EINPROGRESS | EAGAIN))) {
        perror("L2CAP connect");
        goto finish;
      }
    }

    write(s, "hello", 5);
    perror("write");
    close(s);

finish:
    /* Disconnect
    err = hci_disconnect(sock, handle, HCI_CONNECTION_TERMINATED, 10000);
    if (err) {
      perror("disconnect");
      close( sock );
      exit(1);
    }*/

//    close( sock );
    return 0;
}
