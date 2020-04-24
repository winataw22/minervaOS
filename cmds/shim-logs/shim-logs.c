#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <fcntl.h>
#include <sys/epoll.h>
#include <sys/types.h>
#include <sys/stat.h>
#include "shim-logs.h"

void diep(char *str) {
    perror(str);
    exit(EXIT_FAILURE);
}

int main() {
    printf("[+] initializing shim-logs\n");

    //
    // container object
    //
    container_t *container;

    if(!(container = container_init()))
        diep("container");

    if(!(container_load(container))) {
        fprintf(stderr, "[-] could not load configuration\n");
        exit(EXIT_FAILURE);
    }

    //
    // debug file backend
    //

    #if 0
    file_t *lo;
    if(!(lo = file_new("/tmp/log-stdout")))
        exit(EXIT_FAILURE);

    file_t *le;
    if(!(le = file_new("/tmp/log-stderr")))
        exit(EXIT_FAILURE);

    log_attach(container->logout, lo, lo->write);
    log_attach(container->logerr, le, le->write);
    #endif

    //
    //
    //

    //
    // initialize async
    //
    struct epoll_event event;
    struct epoll_event *events = NULL;
    int evfd;

    memset(&event, 0, sizeof(struct epoll_event));

    if((evfd = epoll_create1(0)) < 0)
        diep("epoll_create1");

    event.data.fd = 3;
    event.events = EPOLLIN;

    if(epoll_ctl(evfd, EPOLL_CTL_ADD, 3, &event) < 0)
        diep("epoll_ctl");

    event.data.fd = 4;
    event.events = EPOLLIN;

    if(epoll_ctl(evfd, EPOLL_CTL_ADD, 4, &event) < 0)
        diep("epoll_ctl");

    if(!(events = calloc(MAXEVENTS, sizeof(event))))
        diep("calloc");

    //
    // notify caller we are ready
    //
    container_ready(container);

    //
    // async fetching logs
    //
    while(1) {
        int n = epoll_wait(evfd, events, MAXEVENTS, -1);

        if(n < 0)
            diep("epoll_wait");

        for(int i = 0; i < n; i++) {
            struct epoll_event *ev = events + i;

            if(ev->events & EPOLLIN) {
                log_t *target = NULL;

                if(ev->data.fd == container->logout->fd)
                    target = container->logout;

                if(ev->data.fd == container->logerr->fd)
                    target = container->logerr;

                // printf("[+] reading fd: %d\n", target->fd);
                stream_read(target->fd, target->stream);

                char *line;
                while((line = stream_line(target->stream)))
                    log_dispatch(target, line);

                if(stream_remain(target->stream) == 0) {
                    // printf("[+] recall stream buffer\n");
                    stream_recall(target->stream);
                }
            }
        }
    }

    return 0;
}
