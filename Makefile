export THIS := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

ALL: sgxhttpd


export CC=gcc
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

build/lib/libphp.a: pkg/php-8.1.4.tar.gz build/lib/libz.a build/lib/liblzma.a build/lib/libxml2.a
	mkdir -p build &&\
	cd build &&\
		tar xvf ../pkg/php-8.1.4.tar.gz &&\
	cd php-8.1.4 &&\
	./configure \
		--prefix=/usr/ \
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
	$(MAKE) &&\
	$(MAKE) install\

build/lib/shiv.o: shiv/main.c
	gcc -fPIC -c $^ -o $@

sgxhttpd: .PHONY build/lib/libphp.a build/lib/shiv.o
	export CGO_CFLAGS_ALLOW=".*" &&\
	export CGO_LDFLAGS_ALLOW=".*" &&\
	/usr/local/go/bin/go build -a

.PHONY:
