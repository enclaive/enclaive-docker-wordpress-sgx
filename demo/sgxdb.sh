#!/bin/sh

/edb &

timeout 60 sh -c 'until nc -z localhost 8080; do sleep 1; done'

echo "EDB started up, initializing manifest"

curl -sk https://localhost:8080/manifest --data-binary @- << EOF
{
    "sql": [
        "CREATE USER root@localhost IDENTIFIED BY 'root'",
        "CREATE USER root@'%' IDENTIFIED BY 'root'",
        "GRANT ALL ON *.* TO root WITH GRANT OPTION"
    ]
}
EOF

echo "Manifest successfully uploaded"

tail -f /dev/null
