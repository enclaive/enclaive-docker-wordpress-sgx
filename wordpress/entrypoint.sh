#!/bin/sh
set -e

/opt/mystikos/bin/myst exec-sgx --strace-failing /rootfs  /sgxhttpd --memory-size 4G
