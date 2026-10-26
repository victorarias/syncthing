#!/usr/bin/env python3
# Copyright (C) 2026 The attn Authors.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at https://mozilla.org/MPL/2.0/.

"""Deny filesystem writes except in the two explicit engine paths (Linux Landlock)."""
import ctypes,os,sys
libc=ctypes.CDLL(None,use_errno=True)
class Ruleset(ctypes.Structure):_fields_=[('handled_access_fs',ctypes.c_uint64)]
class PathBeneath(ctypes.Structure):_pack_=1;_fields_=[('allowed_access',ctypes.c_uint64),('parent_fd',ctypes.c_int32)]
def check(value):
 if value<0:raise OSError(ctypes.get_errno(),os.strerror(ctypes.get_errno()))
 return value
abi=check(libc.syscall(444,0,0,1));rights=sum(1<<b for b in [1,4,5,6,7,8,9,10,11,12,13,14])
ruleset=Ruleset(rights);fd=check(libc.syscall(444,ctypes.byref(ruleset),ctypes.sizeof(ruleset),0))
for path in sys.argv[1:3]:
 pathfd=os.open(path,os.O_PATH|os.O_CLOEXEC);rule=PathBeneath(rights,pathfd);check(libc.syscall(445,fd,1,ctypes.byref(rule),0));os.close(pathfd)
check(libc.prctl(38,1,0,0,0));check(libc.syscall(446,fd,0));os.close(fd)
os.execv(sys.argv[3],sys.argv[3:])
