#FROM enclaive/debug-gramine:latest
FROM enclaive/gramine-os:latest

RUN apt-get update &&\
    apt-get install -y bash curl file golang libarchive-tools libargon2-1 libargon2-dev libbz2-dev libcurl4-openssl-dev libkrb5-dev liblzma-dev libmariadb-dev libonig-dev libonig5 libpng-dev libpng16-16 libprotobuf-c1 libreadline-dev libsqlite3-dev libssl-dev libxml2 libxml2-dev libz-dev libzip-dev libzip4 make netcat-openbsd patch wget zip zlib1g zlib1g-dev &&\
    rm -rf /var/lib/apt/lists/*

RUN wget -P /bin https://github.com/edgelesssys/era/releases/latest/download/era &&\
    chmod +x /bin/era

RUN wget https://www.php.net/distributions/php-8.1.4.tar.gz &&\
    tar xvf php-8.1.4.tar.gz &&\
    rm php-8.1.4.tar.gz

RUN cd php-8.1.4 &&\
	./configure \
        --prefix=/usr/ \
        --enable-dba \
        --enable-fpm \
        --enable-gd \
        --enable-mysqlnd \
        --with-password-argon2 \
        --with-bz2 \
        --with-curl \
        --with-kerberos \
        --with-mysqli=mysqlnd \
        --with-openssl \
        --with-pdo-mysql=mysqlnd \
        --with-pdo-sqlite \
        --with-readline \
		--enable-embed=static \
        --with-zip \
        --with-zlib &&\
    sed -e 's/#define PHP_CAN_SUPPORT_PROC_OPEN 1//g' -i ./main/php_config.h &&\
    sed -e 's/#define HAVE_FORK 1//g' -i ./main/php_config.h &&\
    sed -e 's/#define HAVE_RFORK 1//g' -i ./main/php_config.h &&\
	make -j &&\
    make install &&\
    cd .. &&\
    rm -rf php-8.1.4

WORKDIR /app

COPY wordpress /app/

RUN \
    wget -q https://github.com/WordPress/WordPress/archive/refs/tags/5.9.3.zip -O WP_5.9.3.zip &&\
    unzip WP_5.9.3.zip &&\
    rm WP_5.9.3.zip && \
    patch -p1 -d WordPress-5.9.3/ < wordpress_5.9.3.diff &&\
    mv WordPress-5.9.3/ wordpress/ &&\
#    mv wp-config.php wordpress/ &&\
    zip -rm app.zip wordpress/

COPY webserver /app/

RUN \
	export CGO_CFLAGS_ALLOW=".*" &&\
	export CGO_LDFLAGS_ALLOW=".*" &&\
	go build -a &&\
# create sgxphp manifest &&\
    gramine-sgx-gen-private-key &&\
    gramine-manifest -Dlog_level=error -Darch_libdir=/lib/x86_64-linux-gnu php.manifest.template php.manifest &&\
    gramine-sgx-sign --manifest php.manifest --output php.manifest.sgx


VOLUME "/data"
ENTRYPOINT ["/app/entrypoint.sh"]

# ports
EXPOSE 80 443
