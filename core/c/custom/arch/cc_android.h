#ifndef LWIP_ARCH_CC_ANDROID_H
#define LWIP_ARCH_CC_ANDROID_H

/* Include some files for defining library routines */
#include <sys/time.h>
#include <endian.h>
#include <stdlib.h>
#include <inttypes.h>

#include <lwip/opt.h>

#include <android/log.h>

#define LWIP_TIMEVAL_PRIVATE 0

/* Compiler hints for packing structures */
#define PACK_STRUCT_FIELD(x) x
#define PACK_STRUCT_STRUCT __attribute__((packed))
#define PACK_STRUCT_BEGIN
#define PACK_STRUCT_END

#define MY_ANDROID_LOG_TAG  "Trojan-TUN-lwIP"
#define MY_ANDROID_PRINT(...)  __android_log_print(ANDROID_LOG_ERROR, MY_ANDROID_LOG_TAG, ## __VA_ARGS__)

/* Plaform specific diagnostic output */
#define LWIP_PLATFORM_DIAG(...)   do { \
  MY_ANDROID_PRINT("%s/%s(%d): ", __FILE__, __func__, __LINE__); \
  MY_ANDROID_PRINT __VA_ARGS__; \
  MY_ANDROID_PRINT("\n"); \
} while(0)

#define LWIP_PLATFORM_ASSERT(...) do { \
  MY_ANDROID_PRINT("[Assert]%s/%s(%d): %s\n", __FILE__, __func__, __LINE__, __VA_ARGS__); \
  abort(); \
} while(0)

/*
struct sio_status_s;
typedef struct sio_status_s sio_status_t;
#define sio_fd_t sio_status_t*
#define __sio_fd_t_defined
*/

#endif /* LWIP_ARCH_CC_ANDROID_H */
