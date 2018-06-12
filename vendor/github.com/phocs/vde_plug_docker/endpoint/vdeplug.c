/*
 * vdeplug: Allows to connect a device tap to a VDE network
 * Copyright (C) 2018 Alessio Volpe, University of Bologna
 * Credit: inspired by vdens
 *         https://github.com/rd235/vdens
 *
 * vdeplug is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; If not, see <http://www.gnu.org/licenses/>.
 *
 */

#define _GNU_SOURCE
#include "vdeplug.h"
#include <poll.h>
#include <stdio.h>
#include <fcntl.h>
#include <stdlib.h>
#include <signal.h>
#include <unistd.h>
#include <net/if.h>
#include <string.h>
#include <pthread.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <libvdeplug.h>
#include <sys/signalfd.h>
#include <linux/if_tun.h>

#define SUCCESS		1
#define FAILURE		0

struct vdeplug_t {
	pthread_mutex_t mutex;
	int plugged;
	char *tap;
	char *url;
};

#define PTHREAD_MUTEX_INIT_LOCKED \
	({pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER; pthread_mutex_lock(&m); m;})

#define pthread_mutex_free(m_ptr) \
		({pthread_mutex_unlock(m_ptr); pthread_mutex_destroy(m_ptr);})

#define pthread_exit_failure(vpd) \
		({vpd->plugged = FAILURE; pthread_mutex_unlock(&vpd->mutex); pthread_exit(NULL);})

static int open_tap(char *name) {
  struct ifreq ifr;
	int fd=-1;
	if((fd = open("/dev/net/tun", O_RDWR | O_CLOEXEC)) < 0)
		return -1;
	memset(&ifr, 0, sizeof(ifr));
	ifr.ifr_flags = IFF_TAP | IFF_NO_PI;
	snprintf(ifr.ifr_name, sizeof(ifr.ifr_name), "%s", name);
	if(ioctl(fd, TUNSETIFF, (void *) &ifr) < 0) {
		close(fd);
		return -1;
	}
  return fd;
}

void *plug2tap(void *arg) {
	struct vdeplug_t *vpd = arg;
	char buf[VDE_ETHBUFSIZE];
	int tapfd, n, i;
	VDECONN *conn;

  if ((tapfd=open_tap(vpd->tap)) == -1)
		pthread_exit_failure(vpd);
  if ((conn=vde_open(vpd->url, "vde_plug_docker", NULL)) == NULL) {
		close(tapfd);
		pthread_exit_failure(vpd);
	}
	pthread_mutex_unlock(&vpd->mutex);

	struct pollfd pfd[] = {
		{-1, POLLIN, 0},
		{tapfd, POLLIN, 0},
		{-1, POLLIN, 0}
	};
	sigset_t mask;
  sigemptyset(&mask);
  sigaddset(&mask, SIGUSR1);
	pthread_sigmask(SIG_BLOCK, &mask, NULL);
	pfd[0].fd = vde_datafd(conn);
	pfd[2].fd = signalfd(-1, &mask, SFD_CLOEXEC);
	while(ppoll(pfd, 3, NULL, &mask) >= 0) {
		if (pfd[0].revents & POLLIN) {
			n = vde_recv(conn, buf, VDE_ETHBUFSIZE, 0);
			if (n == 0) goto terminate;
			write(tapfd, buf, n);
		}
		if(pfd[1].revents & POLLIN) {
			n = read(tapfd, buf, VDE_ETHBUFSIZE);
			if (n == 0) goto terminate;
			vde_send(conn, buf, n, 0);
		}
		if(pfd[2].revents & POLLIN) {
			break;
		}
	}
terminate:
	vde_close(conn);
	close(tapfd);
  pthread_exit(NULL);
}

uintptr_t vdeplug_start(char *tap_name, char *vde_url) {
	pthread_t *th_ptr = malloc(sizeof(pthread_t));
	struct vdeplug_t vpd = {
		PTHREAD_MUTEX_INIT_LOCKED,
		SUCCESS,
		tap_name,
		vde_url
	};
	if(pthread_create(th_ptr, NULL, plug2tap, &vpd) == 0) {
		pthread_mutex_lock(&vpd.mutex);
		pthread_mutex_free(&vpd.mutex);
		if(vpd.plugged != SUCCESS) {
			pthread_join(*th_ptr, NULL);
			free(th_ptr);
			return FAILURE;
		}
	}
	return (uintptr_t)th_ptr;
}

void vdeplug_stop(uintptr_t th_ptr) {
	if(th_ptr != 0) { //if is running
		pthread_kill(*((pthread_t *)th_ptr), SIGUSR1);
		pthread_join(*((pthread_t *)th_ptr), NULL);
	}
	free((pthread_t *)th_ptr);
}
