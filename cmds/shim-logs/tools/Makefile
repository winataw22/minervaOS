SPAWN = shim-spawn
DEBUG = shim-debug

CFLAGS  = -g -W -Wall -O2
LDFLAGS = 
CC = cc

all: $(SPAWN) $(DEBUG)

$(SPAWN): spawn.o
	$(CC) -o $@ $^ $(LDFLAGS)

$(DEBUG): debug.o
	$(CC) -o $@ $^ $(LDFLAGS)

%.o: %.c
	$(CC) -o $@ -c $< $(CFLAGS)

clean:
	rm -fv *.o

mrproper: clean
	rm -fv $(SPAWN) $(DEBUG)


