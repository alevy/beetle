#include <stdlib.h>
#include <time.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/l2cap.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>

int main(int argc, char** argv) {
  struct timespec start;
  struct timespec end;
  int i;
  for (i = 0; i < 10; i++) {
    int l2capSock = socket(PF_BLUETOOTH, SOCK_SEQPACKET, BTPROTO_L2CAP);
    sleep(1);

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

    clock_gettime(CLOCK_REALTIME, &start);
    if(connect(l2capSock, (struct sockaddr*)&sockAddr, sizeof(sockAddr)) < 0) {
      perror("connect");
    }
    clock_gettime(CLOCK_REALTIME, &end);
    printf("%d\n", (end.tv_nsec - start.tv_nsec) / 1000000 + (end.tv_sec - start.tv_sec) * 1000);

    close(l2capSock);
  }

  return 0;
}
