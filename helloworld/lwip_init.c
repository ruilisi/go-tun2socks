#include "lwip/init.h"
#include "lwip/tcp.h"
#include "lwip/timeouts.h"
#include "hello_server.h"
#include <stdio.h>

void lwip_run(void) {
    lwip_init();
    printf("LWIP initialized\n");
    hello_server_start();

    // Simple polling loop
    while (1) {
        sys_check_timeouts();
    }
}

