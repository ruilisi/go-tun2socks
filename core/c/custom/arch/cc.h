#ifdef _WIN32
  // both win32 and win64 are defined here
  #include "cc_windows.h"
#elif defined(__ANDROID__) || defined(ANDROID) || defined(TUN2SOCKS)
  #include "cc_android.h"
#else
  #include "cc_others.h"
#endif
