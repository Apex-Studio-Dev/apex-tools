/* SPDX-License-Identifier: GPL-2.0 OR Apache-2.0 */
/*
 * Drop-in replacement for <linux/ashmem.h>
 * Combines uapi and kernel-space ashmem interface for userspace builds
 * Origin: Android Open Source Project
 */

#ifndef _LINUX_ASHMEM_H
#define _LINUX_ASHMEM_H

#include <linux/ioctl.h>
#include <linux/types.h>

#define ASHMEM_NAME_LEN        256
#define ASHMEM_NAME_DEF        "dev/ashmem"

/* Return values from ASHMEM_PIN: Was the mapping purged while unpinned? */
#define ASHMEM_NOT_PURGED      0
#define ASHMEM_WAS_PURGED      1

/* Return values from ASHMEM_GET_PIN_STATUS: Is the mapping pinned? */
#define ASHMEM_IS_UNPINNED     0
#define ASHMEM_IS_PINNED       1

struct ashmem_pin {
    __u32 offset; /* Offset into region, in bytes (must be page-aligned) */
    __u32 len;    /* Length from offset, in bytes (must be page-aligned) */
};

#define __ASHMEMIOC                     0x77

#define ASHMEM_SET_NAME                 _IOW(__ASHMEMIOC, 1, char[ASHMEM_NAME_LEN])
#define ASHMEM_GET_NAME                 _IOR(__ASHMEMIOC, 2, char[ASHMEM_NAME_LEN])
#define ASHMEM_SET_SIZE                 _IOW(__ASHMEMIOC, 3, size_t)
#define ASHMEM_GET_SIZE                 _IO(__ASHMEMIOC, 4)
#define ASHMEM_SET_PROT_MASK            _IOW(__ASHMEMIOC, 5, unsigned long)
#define ASHMEM_GET_PROT_MASK            _IO(__ASHMEMIOC, 6)
#define ASHMEM_PIN                      _IOW(__ASHMEMIOC, 7, struct ashmem_pin)
#define ASHMEM_UNPIN                    _IOW(__ASHMEMIOC, 8, struct ashmem_pin)
#define ASHMEM_GET_PIN_STATUS           _IO(__ASHMEMIOC, 9)
#define ASHMEM_PURGE_ALL_CACHES         _IO(__ASHMEMIOC, 10)

/* Optional: for 32-bit userspace on 64-bit kernel */
#ifdef CONFIG_COMPAT
#include <linux/compat.h>
#define COMPAT_ASHMEM_SET_SIZE          _IOW(__ASHMEMIOC, 3, compat_size_t)
#define COMPAT_ASHMEM_SET_PROT_MASK     _IOW(__ASHMEMIOC, 5, unsigned int)
#endif

#endif /* _LINUX_ASHMEM_H */
