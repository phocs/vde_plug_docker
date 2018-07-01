#ifndef VDEPLUG_H
#define VDEPLUG_H

#include <stdint.h>

uintptr_t vdeplug_join(char *tap_name, char *vde_url);
void vdeplug_leave(uintptr_t th);

#endif
