#!/bin/sh
set -e

/opt/mystikos/bin/myst exec-sgx /rootfs  /sgxhttpd --memory-size 2G
