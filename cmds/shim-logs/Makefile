EXEC = shim-logs

CFLAGS  = -g -W -Wall -O2
LDFLAGS = -Wl,-Bstatic -lhiredis -ljansson -Wl,-Bdynamic
CC = cc

SRC=$(wildcard *.c)
OBJ=$(SRC:.c=.o)

all: $(EXEC)

musl: CFLAGS = -g -W -Wall -O2 -march=x86-64
musl: LDFLAGS = -static -lhiredis -ljansson
musl: CC = x86_64-pc-linux-musl-gcc
musl: clean $(EXEC)

$(EXEC): $(OBJ)
	$(CC) -o $@ $^ $(LDFLAGS)

%.o: %.c
	$(CC) -o $@ -c $< $(CFLAGS)

clean:
	rm -fv *.o

mrproper: clean
	rm -fv $(EXEC)


