CC := gcc
CFLAGS += `pkg-config --cflags bluez libuv`
LDFLAGS += `pkg-config --libs bluez libuv`

all: test

clean:
	rm -f test
