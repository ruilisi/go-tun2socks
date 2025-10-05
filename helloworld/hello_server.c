#include "lwip/tcp.h"
#include <string.h>
#include <stdio.h>

static err_t hello_accept(void *arg, struct tcp_pcb *pcb, err_t err);
static err_t hello_recv(void *arg, struct tcp_pcb *pcb, struct pbuf *p, err_t err);

void hello_server_start(void) {
    struct tcp_pcb *pcb = tcp_new();
    tcp_bind(pcb, IP_ANY_TYPE, 8080);
    pcb = tcp_listen(pcb);
    tcp_accept(pcb, hello_accept);
    printf("Hello server listening on port 8080\n");
}

static err_t hello_accept(void *arg, struct tcp_pcb *pcb, err_t err) {
    tcp_recv(pcb, hello_recv);
    return ERR_OK;
}

static err_t hello_recv(void *arg, struct tcp_pcb *pcb, struct pbuf *p, err_t err) {
    if (!p) {
        tcp_close(pcb);
        return ERR_OK;
    }

    const char *msg = "Hello, World!\n";
    tcp_write(pcb, msg, strlen(msg), TCP_WRITE_FLAG_COPY);
    tcp_output(pcb);

    pbuf_free(p);
    tcp_close(pcb);
    return ERR_OK;
}

