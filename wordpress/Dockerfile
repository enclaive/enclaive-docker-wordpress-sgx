#FROM enclaive/debug-gramine:latest
FROM ubuntu:20.04

RUN export DEBIAN_FRONTEND=noninteractive &&\
    apt-get update &&\
    apt-get install -y build-essential wget git autoconf libtool python python2 python2-dev \
    re2c bison vim less netcat-openbsd curl mysql-client pkg-config musl-tools \
    zip &&\
    rm -rf /var/lib/apt/lists/* &&\
    wget https://go.dev/dl/go1.17.9.linux-amd64.tar.gz &&\
    tar -C /usr/local/ -xzf go1.17.9.linux-amd64.tar.gz &&\
    rm go1.17.9.linux-amd64.tar.gz &&\
    wget -P /bin https://github.com/edgelesssys/era/releases/latest/download/era &&\
    chmod +x /bin/era &&\
#mystikos &&\
    echo 'deb [arch=amd64] https://download.01.org/intel-sgx/sgx_repo/ubuntu focal main' | tee /etc/apt/sources.list.d/intel-sgx.list &&\
    wget -qO - https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | apt-key add - &&\
    echo "deb [arch=amd64] https://packages.microsoft.com/ubuntu/20.04/prod focal main" | tee /etc/apt/sources.list.d/msprod.list &&\
    wget -qO - https://packages.microsoft.com/keys/microsoft.asc | apt-key add - &&\
    apt update &&\
    apt -y install libsgx-enclave-common libsgx-dcap-ql libsgx-dcap-ql-dev libsgx-quote-ex az-dcap-client libmbedtls-dev &&\
    export LATEST='0.8.0' &&\
    export RELEASE="Ubuntu-2004_mystikos-${LATEST}-x86_64" &&\
    curl -sSL --ssl https://github.com/deislabs/mystikos/releases/download/v${LATEST}/${RELEASE}.deb -O &&\
    dpkg -i ${RELEASE}.deb &&\
    rm ${RELEASE}.deb &&\
    rm -rf /var/lib/apt/lists/*

RUN \
    mkdir /staging &&\
    mkdir -p /src && cd /src/ &&\
    export CC=musl-gcc &&\
    export LD=musl-gcc &&\
#build zlib &&\
    wget https://zlib.net/zlib-1.2.12.tar.gz &&\
    tar xvf zlib-1.2.12.tar.gz &&\
    cd zlib-1.2.12/ &&\
    ./configure --static --prefix=/staging &&\
    make -j &&\
    make install &&\
    cd .. &&\
#build libxml2 &&\
    git clone https://gitlab.gnome.org/GNOME/libxml2.git &&\
    cd libxml2 &&\
    ./autogen.sh \
     --enable-static \
     --disable-shared \
     --prefix=/staging\
     --without-python \
     --with-zlib=/staging/ &&\
     make -j &&\
     make install &&\
     cd .. &&\
     rm -rf libxml2


RUN \
    mkdir -p /src && cd /src/ &&\
    export PKG_CONFIG_PATH=/staging/lib/pkgconfig/ &&\
    export CFLAGS=-I/staging/include &&\
    export CC=musl-gcc &&\
    export LD=musl-gcc &&\
#build php &&\
    wget https://www.php.net/distributions/php-8.1.4.tar.gz &&\
    tar xvf php-8.1.4.tar.gz &&\
    rm php-8.1.4.tar.gz &&\
	cd php-8.1.4 &&\
	./configure \
        --prefix=/staging/ \
        --disable-all \
        --enable-dba \
        --enable-mysqlnd \
        --with-mysqli=mysqlnd \
        --enable-pdo \
        --with-pdo-mysql=mysqlnd \
		--enable-embed=static \
        &&\
    sed -e 's/#define PHP_CAN_SUPPORT_PROC_OPEN 1//g' -i ./main/php_config.h &&\
    sed -e 's/#define HAVE_FORK 1//g' -i ./main/php_config.h &&\
    sed -e 's/#define HAVE_RFORK 1//g' -i ./main/php_config.h &&\
	make -j &&\
    make install &&\
    cd .. &&\
    rm -rf php-8.1.4


COPY wordpress_5.9.3.diff /src/
COPY wp-config.php /src/
RUN \
    cd /src &&\
    wget -q https://github.com/WordPress/WordPress/archive/refs/tags/5.9.3.zip -O WP_5.9.3.zip &&\
    unzip WP_5.9.3.zip &&\
    rm WP_5.9.3.zip && \
    patch -p1 -d WordPress-5.9.3/ < /src/wordpress_5.9.3.diff &&\
    mv WordPress-5.9.3/ wordpress/ &&\
    mv /src/wp-config.php wordpress/

COPY . /app/
WORKDIR /app

RUN \
    export CC=musl-gcc &&\
    export LD=musl-gcc &&\
	export CGO_CFLAGS_ALLOW=".*" &&\
	export CGO_LDFLAGS_ALLOW=".*" &&\
	/usr/local/go/bin/go build -a -linkshared


RUN \
    rm -rf wordpress &&\
    mv /src/wordpress /app/wordpress &&\
    mkdir -p /app/data/ &&\
    /opt/mystikos/bin/myst mkcpio /app/ /rootfs



VOLUME "/data"
ENTRYPOINT ["/app/entrypoint.sh"]

# ports
EXPOSE 80 443
