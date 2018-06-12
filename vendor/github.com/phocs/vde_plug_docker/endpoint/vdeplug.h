#ifndef VDEPLUG_H
#define VDEPLUG_H

#include <stdint.h>

uintptr_t vdeplug_start(char *tap_name, char *vde_url);

void vdeplug_stop(uintptr_t th);

#endif
