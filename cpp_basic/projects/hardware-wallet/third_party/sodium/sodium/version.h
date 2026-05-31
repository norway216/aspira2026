#ifndef sodium_version_H
#define sodium_version_H

#include "export.h"

#define SODIUM_VERSION_STRING "1.0.18"

#define SODIUM_LIBRARY_VERSION_MAJOR 23
#define SODIUM_LIBRARY_VERSION_MINOR 3

#ifdef __cplusplus
extern "C" {
#endif

int sodium_library_version_major(void);
int sodium_library_version_minor(void);
const char *sodium_version_string(void);

#ifdef __cplusplus
}
#endif

#endif
