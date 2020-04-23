#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <string.h>
#include <hiredis/hiredis.h>

void diep(char *str) {
    perror(str);
    exit(EXIT_FAILURE);
}

int main() {
    int stdo[2], stde[2], lock[2];
    int skip[2];

    printf("[+] spawn: garbage pipes\n");
    if(pipe(skip))
        diep("skip");

    if(pipe(skip))
        diep("skip");

    printf("[+] spawn: opening real pipes\n");
    if(pipe(stdo))
        diep("stdout");

    if(pipe(stde))
        diep("stderr");

    if(pipe(lock))
        diep("lock");

    printf("[+] spawn: forking\n");
    pid_t p = fork();
    if(p == 0) {
        printf("[+] spawn: fork: changing fd\n");
        dup2(stdo[0], 3);
        dup2(stde[0], 4);
        dup2(lock[1], 5);

        printf("[+] spawn: fork: closing pipes reader\n");
        close(stdo[1]);
        close(stde[1]);
        close(lock[0]);

        printf("[+] spawn: fork: setting environment variables\n");
        setenv("CONTAINER_ID", "debug", 0);
        setenv("CONTAINER_NAMESPACE", "maxux", 0);

        printf("[+] spawn: fork: executing shim-logs\n");
        if(execlp("../shim-logs", "../shim-logs", (char *) NULL) < 0)
            diep("execlp");

    } else {
        pid_t px = fork();

        if(px == 0) {
            printf("[+] spawn: waiting for lock\n");

            char buff[32];
            if(read(lock[0], buff, sizeof(buff)) < 0)
                perror("read");

            printf("[+] spawn: starting real process\n");
            dup2(stdo[1], 1);
            dup2(stde[1], 2);

            execlp("./shim-debug", "./shim-debug", (char *) NULL);
        }

        printf("[+] spawn (new): waiting for shim-debug [%d] to finish\n", px);
        int status;
        waitpid(px, &status, 0);

        printf("[+] spawn (new): shim-debug done, status: %d\n", WEXITSTATUS(status));
    }

    printf("[+] spawn: waiting for shim-logs [%d] to finish\n", p);
    int status;
    waitpid(p, &status, 0);

    printf("[+] spawn: shim-logs done, status: %d\n", WEXITSTATUS(status));


    return 0;
}
