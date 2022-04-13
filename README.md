sgxphp
========

a php/http server for running inside a TEE enclave.

 - does not fork (runs php in single heap)
 - wordpress works fine except file uploads are borked

TODO:

 - expose local attestation as php functions
 - implement some sort of enclaved VFS for file uploads
 - to match edb workflow we need to patch wordpress to be able to upload an mtls cert in the installer
