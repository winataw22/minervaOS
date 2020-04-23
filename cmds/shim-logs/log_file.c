#include <stdio.h>
#include "shim-logs.h"

int file_write(void *_self, char *line, int len) {
    file_t *self = (file_t *) _self;

    if(fwrite(line, len, 1, self->fp) != (size_t) len)
        return 1;

    fflush(self->fp);

    return 0;
}

file_t *file_new(char *path) {
    file_t *backend;

    if(!(backend = calloc(sizeof(file_t), 1)))
        diep("calloc");

    if(!(backend->fp = fopen(path, "w")))
        diep("fopen");

    backend->write = file_write;

    return backend;
}

