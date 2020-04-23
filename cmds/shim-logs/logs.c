#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "shim-logs.h"

log_t *log_new(int fd) {
    log_t *log;

    if(!(log = malloc(sizeof(log_t))))
        return NULL;

    log->backlen = 0;
    log->backends = malloc(log->backlen * sizeof(void **));
    log->writers = malloc(log->backlen * sizeof(void **));
    log->fd = fd;

    if(!(log->stream = stream_new(4096)))
        return NULL;

    return log;
}

log_t *log_attach(log_t *log, void *backend, int (*writer)(void *, char *, int)) {
    log->backlen += 1;

    if(!(log->backends = realloc(log->backends, log->backlen * sizeof(void **))))
        return NULL;

    if(!(log->writers = realloc(log->writers, log->backlen * sizeof(void **))))
        return NULL;

    log->backends[log->backlen - 1] = backend;
    log->writers[log->backlen - 1] = writer;

    return log;
}

log_t *log_dispatch(log_t *log, char *line) {
    size_t length = strlen(line);
    // printf("[+] dispatcher: %s", line);

    // calling each writers
    for(size_t i = 0; i < log->backlen; i++)
        log->writers[i](log->backends[i], line, length);

    return log;
}


