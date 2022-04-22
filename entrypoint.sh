#!/bin/sh
set -e

(
    export AESM_PATH=/opt/intel/sgx-aesm-service/aesm
    export LD_LIBRARY_PATH=/opt/intel/sgx-aesm-service/aesm
    cd /opt/intel/sgx-aesm-service/aesm
    /opt/intel/sgx-aesm-service/aesm/linksgx.sh
    /bin/mkdir -p /var/run/aesmd/
    /bin/chown -R aesmd:aesmd /var/run/aesmd/
    /bin/chmod 0755 /var/run/aesmd/
    /bin/chown -R aesmd:aesmd /var/opt/aesmd/
    /bin/chmod 0750 /var/opt/aesmd/
    /opt/intel/sgx-aesm-service/aesm/aesm_service
)

gramine-sgx-get-token --output php.token --sig php.sig
gramine-sgx php
