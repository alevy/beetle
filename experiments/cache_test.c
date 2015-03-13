#include <stdlib.h>
#include <sys/un.h>
#include <time.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/l2cap.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>

void go(int);

int main(int argc, char** argv) {
  int i;
  for (i = 0; i < 10; i++) {
    int pid;
    if ((pid = fork()) == 0) {
      go(i);
      return 0;
    }
  }
  sleep(10);
  return 0;
}

void go(int pid) {
  static char *pth = "/tmp/babel.sock";


  int l2capSock = socket(AF_UNIX, SOCK_SEQPACKET, 0);

  struct sockaddr_un sockAddr;
  memset(&sockAddr, 0, sizeof(sockAddr));
  sockAddr.sun_family = AF_UNIX;
  memcpy(sockAddr.sun_path, pth, strlen(pth));

  if(connect(l2capSock, (struct sockaddr*)&sockAddr, sizeof(sockAddr)) < 0) {
    perror("connect");
    exit(1);
  }

  char req[3] = { 0x0A, 0x06, 0x00 };
  char buf[48];

  struct timespec start;
  struct timespec end;

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

    printf("%d %d\n", pid, (end.tv_nsec - start.tv_nsec) / 1000000 + (end.tv_sec - start.tv_sec) * 1000);
  }
  close(l2capSock);
}

