#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <fcntl.h>
#include <sys/epoll.h>
#include <sys/types.h>
#include <sys/stat.h>
#include "shim-logs.h"
#include "url_parser.h"

int redis_write(void *_self, char *line, int len) {
    (void) len;
    redis_t *self = (redis_t *) _self;
    redisReply *reply;

    if(!(reply = redisCommand(self->conn, "PUBLISH %s %s", self->channel, line)))
        diep("redis");

    freeReplyObject(reply);

    return 0;
}

redis_t *redis_new(char *host, int port, char *channel) {
    redis_t *backend;
    struct timeval timeout = { 2, 0 };

    printf("[+] redis backend: [%s:%d / %s]\n", host, port, channel);

    if(!(backend = calloc(sizeof(redis_t), 1)))
        diep("calloc");

    if(!(backend->conn = redisConnectWithTimeout(host, port, timeout)))
        diep("redis");

    if(backend->conn->err) {
        printf("redis: %s\n", backend->conn->errstr);
        return NULL;
    }

    backend->channel = strdup(channel);
    backend->write = redis_write;

    return backend;
}

static int redis_attach(const char *url, log_t *target) {
    int port = 6379;
    struct parsed_url *purl;
    redis_t *redis;

    if(!(purl = parse_url(url)))
        return 1;

    if(purl->port)
        port = atoi(purl->port);

    if(!(redis = redis_new(purl->host, port, purl->path)))
        return 1;

    parsed_url_free(purl);
    log_attach(target, redis, redis->write);

    return 0;
}

int redis_extract(container_t *c, json_t *root) {
    json_t *sout = json_object_get(root, "stdout");
    json_t *serr = json_object_get(root, "stderr");

    if(!json_is_string(sout) || !json_is_string(serr))
        return 1;

    redis_attach(json_string_value(sout), c->logout);
    redis_attach(json_string_value(serr), c->logerr);

    return 0;
}
