# EdgelessDB VS MariaDB

## Building and Running

```bash
docker-compose up
```

## Demonstrating encryption

- EdgelessDB is running at port `3306`
- MariaDB uses port `3307`

To find the storage files, use `fswatch` while creating databases and tables:

```bash
fswatch -rtuxa sgxdb/#rocksdb/ mariadb/#rocksdb/
```

To create databases and tables, execute:

```bash
mysql -h127.0.0.1 -proot -uroot -P3306 < init.sql
mysql -h127.0.0.1 -proot -uroot -P3307 < init.sql
```

Output should be similar to this:

```
# edb
TIMESTAMP CWD/sgxdb/#rocksdb/000015.log Updated
TIMESTAMP CWD/sgxdb/#rocksdb/000015.log Updated

# mariadb
TIMESTAMP CWD/mariadb/#rocksdb/000026.log Updated
TIMESTAMP CWD/mariadb/#rocksdb/000026.log Updated
```

We can now watch these files using `tail` and as a hexdump with `xxd`:

```bash
tail -q -n 0 -f sgxdb/#rocksdb/000015.log mariadb/#rocksdb/000026.log | xxd -c 16
```

...and insert some test data:

```bash
mysql -h127.0.0.1 -proot -uroot -P3306 < data.sql
mysql -h127.0.0.1 -proot -uroot -P3307 < data.sql
```

