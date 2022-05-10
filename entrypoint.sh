#!/bin/sh

set -e

# aesmd proxy is required for gramine
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

#setup edb, see https://docs.edgeless.systems/edgelessdb/#/getting-started/quickstart-sgx

timeout 60 sh -c 'until nc -z $0 $1; do sleep 1; done' edb 8080

wget https://github.com/edgelesssys/edgelessdb/releases/latest/download/edgelessdb-sgx.json
era -c edgelessdb-sgx.json -h edb:8080 -output-root /app/edb.pem -skip-quote

cat - > manifest.json <<EOF
{
    "sql": [
        "CREATE USER root@localhost IDENTIFIED BY 'root'",
        "CREATE USER root@'%' IDENTIFIED BY 'root'",
        "GRANT ALL ON *.* TO root WITH GRANT OPTION",
        "CREATE DATABASE wordpress"
    ]
}
EOF
curl --cacert edb.pem --data-binary @manifest.json https://edb:8080/manifest

cp edb.pem /usr/local/share/ca-certificates/
update-ca-certificates

gramine-sgx-get-token --output php.token --sig php.sig
gramine-sgx php
