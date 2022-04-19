export THIS := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

ALL: build sgxhttpd


export CC=musl-gcc
export PKG_CONFIG_PATH=$(THIS)/build/lib/pkgconfig/
export CFLAGS=\
	-fPIC \
	-g \
	-I$(THIS)/build/include \

export LDFLAGS=\
	-g \
	-L$(THIS)/build/lib

env:
	env

clean:
	rm -rf build

build:
	mkdir build

build/lib/libz.a: pkg/zlib-1.2.12.tar.gz
	cd build &&\
	tar xvf ../pkg/zlib-1.2.12.tar.gz &&\
	cd zlib-1.2.12 &&\
	./configure --prefix=$$PWD/../ &&\
	$(MAKE) &&\
	$(MAKE) install

build/lib/liblzma.a : pkg/xz-5.2.5.tar.gz
	cd build &&\
	tar xvf ../pkg/xz-5.2.5.tar.gz &&\
	cd xz-5.2.5 &&\
	./configure --prefix=$$PWD/../ --disable-shared --enable-static --disable-xz --disable-xzdec &&\
	$(MAKE) &&\
	$(MAKE) install\

build/lib/libxml2.a : pkg/libxml2-2.9.13.tgz
	cd build &&\
	tar xvf ../pkg/libxml2-2.9.13.tgz &&\
	cd libxml2 &&\
	autoreconf -fiv &&\
	./configure --prefix=$$PWD/../ --without-threads --disable-shared --enable-static &&\
	$(MAKE) &&\
	$(MAKE) install\

build/lib/libmusl.a: pkg/musl-1.2.3.tar.gz
	cd build &&\
	tar xvf ../pkg/musl-1.2.3.tar.gz &&\
	cd musl-1.2.3 &&\
	./configure &&\
	$(MAKE) &&\
	ar rcs ../lib/libmusl.a \
		obj/src/math/*.lo obj/src/math/x86_64/*.lo \
		obj/src/fenv/*.lo obj/src/fenv/x86_64/*.lo \
		obj/src/string/*.lo \

#		obj/src/errno/*.lo \
#		obj/src/internal/*.lo \
#		obj/src/stdio/__stdio_seek.lo \
#		obj/src/unistd/lseek.lo \
#		obj/src/stdio/stderr.lo \
#		obj/src/stdio/stdout.lo \
#		obj/src/stdio/stdin.lo \
#		obj/src/stdio/__stdio_close.lo \
#		obj/src/stdio/__stdio_read.lo \
#		obj/src/stdio/__stdio_write.lo \
#		obj/src/stdio/__stdout_write.lo \
#		obj/src/signal/x86_64/sigsetjmp.lo \
#		obj/src/signal/sigsetjmp_tail.lo


build/lib/libphp.a: pkg/php-8.1.4.tar.gz build/lib/libz.a build/lib/liblzma.a build/lib/libxml2.a | build/lib/libmusl.a
	cd build &&\
		tar xvf ../pkg/php-8.1.4.tar.gz &&\
	cd php-8.1.4 &&\
	./configure \
		--prefix=$$PWD/../ \
		--disable-all \
		--enable-pdo \
		--enable-static \
		--disable-shared \
		--disable-sapi \
		--disable-cli \
		--disable-cgi \
		--disable-phpdbg \
		--enable-embed=static \
		--enable-dba \
        --enable-mysqlnd \
        --with-mysqli=mysqlnd \
        --with-pdo-mysql=mysqlnd \
        --with-zlib &&\
	sed -e 's/#define HAVE_LIBDL 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_CHROOT 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_LIBCRYPT 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_CRYPT 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_CRYPT_H 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_CRYPT_R 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_DLOPEN 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_DN_EXPAND 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_DN_SKIPNAME 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_RES_NSEARCH 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_RES_SEARCH 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_OPENPTY 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_MMAP 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define PHP_WRITE_STDOUT 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_LIBM 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_LIBRESOLV 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_DLSYM 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_HAVE_ATOLL 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define PHP_HAVE_BUILTIN_CPU_INIT 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define PHP_HAVE_BUILTIN_CPU_SUPPORTS 1//g' -i ./main/php_config.h &&\
	sed -e 's/#define HAVE_SIGSETJMP 1//g' -i ./main/php_config.h &&\
	$(MAKE) &&\
	$(MAKE) install\



build/lib/shiv.o: shiv/main.c
	gcc -fPIC -c $^ -o $@

prebug: ./build/lib/libphp.a ./build/lib/libxml2.a ./build/lib/liblzma.a ./build/lib/libz.a ./build/lib/libmusl.a
	gcc repro/main.c shim.c $^

sgxhttpd: .PHONY build/lib/libphp.a build/lib/shiv.o
	export CGO_CFLAGS_ALLOW=".*" &&\
	export CGO_LDFLAGS_ALLOW=".*" &&\
	ego-go build
	ego sign sgxhttpd

.PHONY:
