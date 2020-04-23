#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <fcntl.h>
#include <sys/types.h>
#include "shim-logs.h"

container_t *container_init() {
    container_t *container;

    if(!(container = calloc(sizeof(container_t), 1)))
        return NULL;

    container->id = getenv("CONTAINER_ID");
    container->namespace = getenv("CONTAINER_NAMESPACE");
    container->lockfd = 5; // lock wait fd
    container->outfd = 3;  // stdout fd
    container->errfd = 4;  // stderr fd

    if(container->id == NULL) {
        fprintf(stderr, "[-] could not find container id\n");
        fprintf(stderr, "[-] ensure your environment is well set\n");
        exit(EXIT_FAILURE);
    }

    if(!(container->logout = log_new(container->outfd)))
        exit(EXIT_FAILURE);

    if(!(container->logerr = log_new(container->errfd)))
        exit(EXIT_FAILURE);

    return container;
}

void container_ready(container_t *container) {
    printf("[+] sending ready signal\n");

    if(write(container->lockfd, "X", 1) != 1)
        perror("write");

    close(container->lockfd);
}

char *container_load_config(char *path) {
    int fd;
    char *buffer;

    if((fd = open(path, O_RDONLY)) < 0)
        return NULL;

    // grabbing length
    off_t end = lseek(fd, 0, SEEK_END);
    lseek(fd, 0, SEEK_SET);

    if(!(buffer = malloc(end)))
        return NULL;

    if(read(fd, buffer, end) != end)
        perror("read");

    close(fd);

    return buffer;
}

static void *json_error(json_t *root, char *message) {
    fprintf(stderr, "[-] json: %s\n", message);
    json_decref(root);
    return NULL;
}

container_t *container_load_parse(container_t *c, json_t *root) {
    if(!json_is_array(root))
        return json_error(root, "expected root array");

    for(size_t i = 0; i < json_array_size(root); i++) {
        json_t *data, *type;

        data = json_array_get(root, i);
        if(!json_is_object(data))
            return json_error(root, "array item not an object");

        type = json_object_get(data, "type");
        if(!json_is_string(type))
            return json_error(root, "type is not a string");

        const char *stype = json_string_value(type);

        if(strcmp(stype, "redis") == 0) {
            json_t *config = json_object_get(data, "data");
            redis_extract(c, config);

        } else {
            // only supporting redis for now
            fprintf(stderr, "[-] config: unsupported <%s> target\n", stype);
        }
    }

    return c;
}

container_t *container_load(container_t *c) {
    char path[512];
    char *buffer;
    json_t *root;
    json_error_t error;

    // setting up config path
    sprintf(path, "%s/%s/%s-logs.json", CONFDIR, c->namespace, c->id);

    printf("[+] loading configuration: %s\n", path);
    if(!(buffer = container_load_config(path)))
        return NULL;

    // parsing json config
    // printf(">> %s\n", buffer);

    if(!(root = json_loads(buffer, 0, &error))) {
        fprintf(stderr, "json error: %d: %s\n", error.line, error.text);
        return NULL;
    }

    free(buffer);

    // fetching data from json and populate config
    return container_load_parse(c, root);
}


