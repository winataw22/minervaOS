#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

char *availables = "ABCDEFGHIKLMOPQRSTUVWXUZabcdefghijklmnopqrstuvwxyz0123456789$#";

char *rn(char *buffer, size_t len) {
    size_t max = strlen(availables);

    for(size_t i = 0; i < len - 1; i++)
        buffer[i] = availables[rand() % max];

    return buffer;
}

int main(void) {
    char buffer[512], rnds[65];
    int written = 0, errors = 0;

    memset(rnds, 0x00, sizeof(rnds));
    srand(time(NULL));

    for(int i = 0; i < 64; i++) {
        written += sprintf(buffer, "[+ %6d]  %s", written, rn(rnds, sizeof(rnds)));
        puts(buffer);
        usleep(60000);

        if(written % 72 == 0) {
            errors += sprintf(buffer, "[- %6d]  %s", errors, rn(rnds, sizeof(rnds)));
            fprintf(stderr, "%s\n", buffer);
        }
    }

    return 0;
}
