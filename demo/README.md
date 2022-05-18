# SGXPHP WordPress Encryption Demonstration

## Building and Running

```bash
docker-compose up
```

## Demonstrating encryption with wordpress

To find the storage files, use `fswatch` while creating databases and tables:

```bash
fswatch -rtuxa sgxdb/#rocksdb/ stddb/#rocksdb/
```

To create databases and tables, run the wordpress installer.

Standard WordPress runs at `8443`. SGX-WordPress listens on `443`. Both use TLS with a self-signed certificate.

Output of `fswatch` should look like this:

```bash
TIMESTAMP CWD/sgxdb/#rocksdb/000015.log Updated
TIMESTAMP CWD/sgxdb/#rocksdb/000015.log Updated

TIMESTAMP CWD/stddb/#rocksdb/000026.log Updated
TIMESTAMP CWD/stddb/#rocksdb/000026.log Updated
```

We can now watch these files using `tail` and as a hexdump with `xxd`:

```bash
tail -q -n 0 -f sgxdb/#rocksdb/000015.log stddb/#rocksdb/000026.log | xxd -c 16
```

... or use `grep` with our data:

```bash
grep -R TestUsername sgxdb/#rocksdb/ stddb/#rocksdb/
> grep: stddb/#rocksdb/000026.log: binary file matches

grep -a TestUsername stddb/#rocksdb/000026.log | xxd
00000000: 0001 0000 0000 0000 0016 8395 e6fa 3301  ..............3.
00000010: 0191 0200 0000 0000 0005 0000 000d 010c  ................
00000020: 0000 0100 0000 0000 0000 0001 690c 5465  ............i.Te
00000030: 7374 5573 6572 6e61 6d65 2200 2450 2442  stUsername".$P$B
00000040: 3437 784d 4d65 6f34 6a62 4b75 2f41 2e74  47xMMeo4jbKu/A.t
00000050: 3455 465a 7552 446e 6a6b 7039 7a2e 0c74  4UFZuRDnjkp9z..t
00000060: 6573 7475 7365 726e 616d 650e 0072 6f6f  estusername..roo
00000070: 7440 726f 6f74 2e72 6f6f 7400 0099 acd3  t@root.root.....
...
```

Access logs are in `sgxdb/access.log` and `stddb/access.log`:

```bash
cat stddb/access.log
... raw data
xxd sgxdb/access.log
... encrypted data
```
