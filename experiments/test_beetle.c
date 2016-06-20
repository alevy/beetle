#include <stdlib.h>
#include <time.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/l2cap.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>
#include <unistd.h>
#include <stdio.h>

int main(int argc, char** argv) {
  int sockfd = socket(AF_INET, SOCK_STREAM, 0);
  if (sockfd < 0) {
    perror("ERROR opening socket");
  }
  struct sockaddr_in serv_addr;
  serv_addr.sin_family = AF_INET;
  serv_addr.sin_addr.s_addr = INADDR_ANY;
  serv_addr.sin_port = htons(1432);
  if (bind(sockfd, (struct sockaddr *) &serv_addr,
           sizeof(serv_addr)) < 0) {
    perror("ERROR on binding");
  }
  listen(sockfd,5);

  int l2capSock = accept(sockfd, NULL, NULL);

  FILE* out = fopen(argv[1], "w+");

  char conn_update[3] = { 0xF0, 0x00, 0x00 };
  char req[3] = { 0x0A, 0x01, 0x00 };
  char buf[48];

  struct timespec start;
  struct timespec end;

  uint16_t interval;
  for (interval = 6; interval <= 800; interval *= 2) {
    conn_update[1] = interval & 0xff;
    conn_update[2] = (interval >> 8) & 0xff;
    if(write(l2capSock, conn_update, 3) < 0) {
      perror("conn_update");
    }
    printf("Wrote\n");
    if(read(l2capSock, buf, 1) < 0) {
      perror("conn_update");
    }
    printf("Read\n");


    int i = 0;
    for (i = 0; i < 30; i++) {
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
      fprintf(out, "%f,%d\n", interval * 1.25, (end.tv_nsec - start.tv_nsec) / 1000000 + (end.tv_sec - start.tv_sec) * 1000);
      sync();
      int skip = rand();
      double skipd = ((double)skip / RAND_MAX) * interval * 1.25 * 1000;
      usleep((long)skipd);
    }
  }

  fclose(out);

  return 0;
}
