#ifndef sodium_export_H
#define sodium_export_H

#if defined(__GNUC__)
# ifdef __ELF__
#  define SODIUM_EXPORT __attribute__((visibility("default")))
#  define SODIUM_EXPORT_WEAK __attribute__((weak))
# else
#  define SODIUM_EXPORT
#  define SODIUM_EXPORT_WEAK
# endif
#else
# define SODIUM_EXPORT
# define SODIUM_EXPORT_WEAK
#endif

#endif
