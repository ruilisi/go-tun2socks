/**
 * @file lwipopts.h
 * @author Ambroz Bizjak <ambrop7@gmail.com>
 * 
 * @section LICENSE
 * 
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 * 3. Neither the name of the author nor the
 *    names of its contributors may be used to endorse or promote products
 *    derived from this software without specific prior written permission.
 * 
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */
#include <stdint.h>

#ifndef LWIP_CUSTOM_LWIPOPTS_H
#define LWIP_CUSTOM_LWIPOPTS_H

#define XSTR(x) STR(x)
#define STR(x) #x

// enable tun2socks logic
#define TUN2SOCKS 1

#define NO_SYS 1
#define LWIP_TIMERS 1

#define LWIP_ARP 0
#define ARP_QUEUEING 0
#define IP_FORWARD 0
#define LWIP_ICMP 1
#define LWIP_RAW 1
#define LWIP_DHCP 0
#define LWIP_AUTOIP 0
#define LWIP_SNMP 0
#define LWIP_IGMP 0
#define LWIP_DNS 0
#define LWIP_UDP 1
#define LWIP_UDPLITE 0
#define LWIP_TCP 1
#define LWIP_CALLBACK_API 1
#define LWIP_NETIF_API 0
#define LWIP_NETIF_LOOPBACK 0
#define LWIP_HAVE_LOOPIF 1
#define LWIP_HAVE_SLIPIF 0
#define LWIP_NETCONN 0
#define LWIP_SOCKET 0
#define PPP_SUPPORT 0
#define LWIP_IPV6 1
#define LWIP_IPV6_MLD 0
#define LWIP_IPV6_AUTOCONFIG 1

// disable checksum checks
#define CHECKSUM_CHECK_IP 0
#define CHECKSUM_CHECK_UDP 0
#define CHECKSUM_CHECK_TCP 0
#define CHECKSUM_CHECK_ICMP 0
#define CHECKSUM_CHECK_ICMP6 0

#define LWIP_CHECKSUM_ON_COPY 1

/* 32bit and 64bit CPU */
#if INTPTR_MAX == INT32_MAX
#warning THIS_IS_32BIT_ENVIRONMENT
#define CPU_WORD_LEN     32
#elif INTPTR_MAX == INT64_MAX
// #warning THIS_IS_64BIT_ENVIRONMENT
#define CPU_WORD_LEN     64
#else
#error "Environment not 32 or 64-bit."
#endif

//#pragma message "The value of CPU_WORD_LEN: " XSTR(CPU_WORD_LEN)

#define CPU_BYTE_LEN     8
#define _KB              (1024)
#define _MB             (1024 * 1024)

#define MEM_ALIGNMENT                  (CPU_WORD_LEN / CPU_BYTE_LEN)
//#pragma message "The value of MEM_ALIGNMENT: " XSTR(MEM_ALIGNMENT)

//#define MEM_SIZE                       (512 * _KB)
#define MEM_SIZE                       (32 * _MB)
#define MEMP_NUM_PBUF                   512
#define PBUF_POOL_SIZE                  512
#define PBUF_POOL_BUFSIZE               1600

#if MEM_SIZE >= (32 * _MB)
#define MEMP_NUM_REASSDATA             150
#elif MEM_SIZE >= (1 * _MB)
#define MEMP_NUM_REASSDATA             100
#elif MEM_SIZE >= (512 * _KB)
#define MEMP_NUM_REASSDATA              80
#elif MEM_SIZE >= (256 * _KB)
#define MEMP_NUM_REASSDATA              40
#elif MEM_SIZE >= (128 * _KB)
#define MEMP_NUM_REASSDATA              20
#else
#define MEMP_NUM_REASSDATA              5
#endif

#define MEMP_NUM_TCP_PCB_LISTEN 5
#define MEMP_NUM_TCP_PCB 1024
#define MEMP_NUM_UDP_PCB 512

#define LWIP_TCP_TIMESTAMPS             1
#define IP_REASS_MAX_PBUFS              (MEMP_NUM_PBUF / 2)


#define TCP_MSS                         1460

#define TCP_CALCULATE_EFF_SEND_MSS      1

#if MEM_SIZE >= (32 * _MB)
#define TCP_WND                         ((64 * _KB) - 1)
#define TCP_SND_BUF                     (64 * _KB)
#elif MEM_SIZE >= (16 * _MB)
#define TCP_WND                         ((64 * _KB) - 1)
#define TCP_SND_BUF                     (64  * _KB)
#elif MEM_SIZE >= (4 * _MB)
#define TCP_WND                         (32  * _KB)
#define TCP_SND_BUF                     (32  * _KB)
#elif MEM_SIZE >= (1 * _MB)
#define TCP_WND                         (16  * _KB)
#define TCP_SND_BUF                     (32  * _KB)
#elif MEM_SIZE >= (512 * _KB)
#define TCP_WND                         ( 8  * _KB)
#define TCP_SND_BUF                     (16  * _KB)
#elif MEM_SIZE >= (128 * _KB)
#define TCP_WND                         ( 8  * _KB)
#define TCP_SND_BUF                     ( 8  * _KB)
#elif MEM_SIZE >= (64 * _KB)  /* MEM_SIZE < 128 _KB  SMALL TCP_MSS XXX */
#undef  TCP_MSS
#define TCP_MSS                         536
#define TCP_WND                         ( 4  * TCP_MSS)
#define TCP_SND_BUF                     ( 4  * TCP_MSS)
#else
#undef  TCP_MSS
#define TCP_MSS                         256
#define TCP_WND                         ( 4  * TCP_MSS)
#define TCP_SND_BUF                     ( 4  * TCP_MSS)
#endif

#if TCP_WND < (4  * TCP_MSS)
#define TCP_WND                         ( 4  * TCP_MSS)
#endif

#define TCP_SND_QUEUELEN                ((4 * (TCP_SND_BUF) + (TCP_MSS - 1))/(TCP_MSS))
#define MEMP_NUM_TCP_SEG                (8 * TCP_SND_QUEUELEN)

#define LWIP_WND_SCALE                  1
#define TCP_RCV_SCALE                   0
#define LWIP_TCP_KEEPALIVE              1

#define LWIP_TCP_SACK_OUT 1
#define LWIP_TCP_MAX_SACK_NUM           8

#define MEM_LIBC_MALLOC 0
#define MEMP_MEM_MALLOC 0

#define SYS_LIGHTWEIGHT_PROT 0
#define LWIP_DONT_PROVIDE_BYTEORDER_FUNCTIONS

// needed on 64-bit systems, enable it always so that the same configuration
// is used regardless of the platform
#define IPV6_FRAG_COPYHEADER 1

// whether we are in debug mode
// modify any golang (*.go) file to take effect
#define TUN2SOCKS_DEBUG 0

#if TUN2SOCKS_DEBUG

#undef LWIP_NOASSERT
#define LWIP_DEBUG 1

#define LWIP_DBG_MIN_LEVEL              LWIP_DBG_LEVEL_ALL
#define LWIP_DBG_TYPES_ON               LWIP_DBG_ON
#define ETHARP_DEBUG                    LWIP_DBG_ON
#define NETIF_DEBUG                     LWIP_DBG_OFF
#define PBUF_DEBUG                      LWIP_DBG_ON
#define ICMP_DEBUG                      LWIP_DBG_ON
#define IGMP_DEBUG                      LWIP_DBG_ON
#define INET_DEBUG                      LWIP_DBG_ON
#define IP_DEBUG                        LWIP_DBG_ON
#define IP_REASS_DEBUG                  LWIP_DBG_ON
#define RAW_DEBUG                       LWIP_DBG_ON
#define MEM_DEBUG                       LWIP_DBG_ON
#define MEMP_DEBUG                      LWIP_DBG_ON
#define SYS_DEBUG                       LWIP_DBG_ON
#define TIMERS_DEBUG                    LWIP_DBG_OFF
#define TCP_DEBUG                       LWIP_DBG_ON
#define TCP_INPUT_DEBUG                 LWIP_DBG_ON
#define TCP_FR_DEBUG                    LWIP_DBG_ON
#define TCP_RTO_DEBUG                   LWIP_DBG_ON
#define TCP_CWND_DEBUG                  LWIP_DBG_ON
#define TCP_WND_DEBUG                   LWIP_DBG_ON
#define TCP_OUTPUT_DEBUG                LWIP_DBG_ON
#define TCP_RST_DEBUG                   LWIP_DBG_ON
#define TCP_QLEN_DEBUG                  LWIP_DBG_ON
#define UDP_DEBUG                       LWIP_DBG_ON
#define TCPIP_DEBUG                     LWIP_DBG_OFF
#define SLIP_DEBUG                      LWIP_DBG_OFF
#define DHCP_DEBUG                      LWIP_DBG_OFF
#define AUTOIP_DEBUG                    LWIP_DBG_OFF
#define ACD_DEBUG                       LWIP_DBG_ON
#define DNS_DEBUG                       LWIP_DBG_ON
#define IP6_DEBUG                       LWIP_DBG_ON
#define DHCP6_DEBUG                     LWIP_DBG_ON

#else // TUN2SOCKS_DEBUG

#undef LWIP_NOASSERT
#define LWIP_DEBUG 0

#define LWIP_DBG_MIN_LEVEL              LWIP_DBG_LEVEL_WARNING
#define LWIP_DBG_TYPES_ON               LWIP_DBG_OFF
#define ETHARP_DEBUG                    LWIP_DBG_OFF
#define NETIF_DEBUG                     LWIP_DBG_OFF
#define PBUF_DEBUG                      LWIP_DBG_ON
#define ICMP_DEBUG                      LWIP_DBG_OFF
#define IGMP_DEBUG                      LWIP_DBG_OFF
#define INET_DEBUG                      LWIP_DBG_OFF
#define IP_DEBUG                        LWIP_DBG_OFF
#define IP_REASS_DEBUG                  LWIP_DBG_OFF
#define RAW_DEBUG                       LWIP_DBG_OFF
#define MEM_DEBUG                       LWIP_DBG_ON
#define MEMP_DEBUG                      LWIP_DBG_ON
#define SYS_DEBUG                       LWIP_DBG_OFF
#define TIMERS_DEBUG                    LWIP_DBG_OFF
#define TCP_DEBUG                       LWIP_DBG_OFF
#define TCP_INPUT_DEBUG                 LWIP_DBG_OFF
#define TCP_FR_DEBUG                    LWIP_DBG_OFF
#define TCP_RTO_DEBUG                   LWIP_DBG_OFF
#define TCP_CWND_DEBUG                  LWIP_DBG_OFF
#define TCP_WND_DEBUG                   LWIP_DBG_OFF
#define TCP_OUTPUT_DEBUG                LWIP_DBG_OFF
#define TCP_RST_DEBUG                   LWIP_DBG_OFF
#define TCP_QLEN_DEBUG                  LWIP_DBG_OFF
#define UDP_DEBUG                       LWIP_DBG_OFF
#define TCPIP_DEBUG                     LWIP_DBG_OFF
#define SLIP_DEBUG                      LWIP_DBG_OFF
#define DHCP_DEBUG                      LWIP_DBG_OFF
#define AUTOIP_DEBUG                    LWIP_DBG_OFF
#define ACD_DEBUG                       LWIP_DBG_OFF
#define DNS_DEBUG                       LWIP_DBG_OFF
#define IP6_DEBUG                       LWIP_DBG_OFF
#define DHCP6_DEBUG                     LWIP_DBG_OFF

#endif // TUN2SOCKS_DEBUG


#define LWIP_STATS 0
#define LWIP_STATS_DISPLAY 0
#define LWIP_PERF 0

#endif // LWIP_CUSTOM_LWIPOPTS_H
