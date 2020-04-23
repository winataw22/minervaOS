#ifndef SHIM_LOGS_H
    #define SHIM_LOGS_H

    #include <hiredis/hiredis.h>
    #include <jansson.h>

    #define MAXEVENTS   64
    #define CONFDIR     "/var/cache/modules/contd/config"
    #define LOGSDIR     "/var/cache/modules/contd/logs"
    // #define CONFDIR     "/tmp/logconf"
    // #define LOGSDIR     "/tmp/logconf"

    //
    // stream
    // contains tools to read and bufferize
    // from low level file descriptor
    //
    typedef struct stream_t {
        char *buffer;
        int length;
        char *reader;
        char *writer;
        char *line;

    } stream_t;

    stream_t *stream_new(int size);
    int stream_remain(stream_t *s);
    int stream_read(int fd, stream_t *s);
    char *stream_line(stream_t *s);
    void stream_recall(stream_t *s);

    //
    // logs
    // contains list of backends for a specific stream
    // eg: stdout stream with 3 backends (2 redis and 1 file)
    //
    typedef struct log_t {
        void **backends;
        int (**writers)(void *, char *, int);
        size_t backlen;
        stream_t *stream;
        int fd;

    } log_t;

    log_t *log_new(int fd);
    log_t *log_attach(log_t *log, void *backend, int (*writer)(void *, char *, int));
    log_t *log_dispatch(log_t *log, char *line);

    //
    // container
    // make the link between containerd process and
    // our logger
    //
    typedef struct container_t {
        char *id;
        char *namespace;
        int lockfd;
        int outfd;
        int errfd;
        log_t *logout;
        log_t *logerr;

    } container_t;

    container_t *container_init();
    void container_ready(container_t *container);
    char *container_load_config(char *path);
    container_t *container_load_parse(container_t *c, json_t *root);
    container_t *container_load(container_t *c);

    //
    // redis
    // specific backend to publish some line into redis
    //
    typedef struct redis_t {
        redisContext *conn;
        char *channel;
        int (*write)(void *self, char *line, int len);

    } redis_t;

    int redis_write(void *_self, char *line, int len);
    redis_t *redis_new(char *host, int port, char *channel);
    int redis_extract(container_t *c, json_t *root);

    //
    // file
    // specific backend to write log line into a file
    //
    typedef struct file_t {
        FILE *fp;
        char *path;
        int (*write)(void *self, char *line, int len);

    } file_t;

    int file_write(void *_self, char *line, int len);
    file_t *file_new(char *path);

    //
    // utilities
    //
    void diep(char *s);

#endif
