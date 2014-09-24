#include <stdlib.h>
#include <time.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/l2cap.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>

int main(int argc, char** argv) {
  int hciSocket = hci_open_dev(0);

  int l2capSock = socket(PF_BLUETOOTH, SOCK_SEQPACKET, BTPROTO_L2CAP);

  struct sockaddr_l2 sockAddr;
  memset(&sockAddr, 0, sizeof(sockAddr));
  sockAddr.l2_family = AF_BLUETOOTH;
  bacpy(&sockAddr.l2_bdaddr, BDADDR_ANY);
  sockAddr.l2_cid = htobs(4);

  if(bind(l2capSock, (struct sockaddr*)&sockAddr, sizeof(sockAddr)) < 0) {
    perror("bind");
  }

  memset(&sockAddr, 0, sizeof(sockAddr));
  sockAddr.l2_family = AF_BLUETOOTH;
  str2ba(argv[1], &sockAddr.l2_bdaddr);
  sockAddr.l2_bdaddr_type = strcmp(argv[2], "random") == 0 ? BDADDR_LE_RANDOM : BDADDR_LE_PUBLIC;
  sockAddr.l2_cid = htobs(4);

  if(connect(l2capSock, (struct sockaddr*)&sockAddr, sizeof(sockAddr)) < 0) {
    perror("connect");
  }

  struct l2cap_conninfo l2capConnInfo;
  int l2capConnInfoLen = sizeof(l2capConnInfo);
  if(getsockopt(l2capSock, SOL_L2CAP, L2CAP_CONNINFO, &l2capConnInfo, &l2capConnInfoLen) < 0) {
    perror("conninfo");
  }
  int hciHandle = l2capConnInfo.hci_handle;

  char req[3] = { 0x0A, 0x06, 0x00 };
  char buf[48];

  struct timespec start;
  struct timespec end;

  uint16_t interval;
  for (interval = 6; interval <= 0x0C80; interval *= 2) {
    if(hci_le_conn_update(hciSocket, l2capConnInfo.hci_handle, interval, interval, 0, 0x03E8, 0) < 0) {
      perror("conn_update");
    }


    int i = 0;
    for (i = 0; i < 10; i++) {
      clock_gettime(CLOCK_REALTIME, &start);
      if(write(l2capSock, req, 3) < 0){
        perror("write");
      }

      int red;
      if((red = read(l2capSock, buf, 48)) < 0){
        perror("read");
      }
      clock_gettime(CLOCK_REALTIME, &end);

      printf("%f,%d\n", interval * 1.25, (end.tv_nsec - start.tv_nsec) / 1000000 + (end.tv_sec - start.tv_sec) * 1000);
      int skip = rand();
      double skipd = ((double)skip / RAND_MAX) * interval * 1.25 * 1000;
      usleep((long)skipd);
    }
  }

  return 0;
}
