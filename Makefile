CFLAGS += `pkg-config --cflags bluez`
LDFLAGS += `pkg-config --libs bluez`

test: test.c
	gcc $(CFLAGS) $(LDFLAGS) -o $@ $<
