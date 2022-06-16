FROM ubuntu:impish AS builder

COPY ./build/packages.txt .

RUN apt-get update \
    && xargs -a packages.txt -r apt-get install -y \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://github.com/edgelesssys/era/releases/latest/download/era -q \
    && chmod +x era

COPY ./build/php_8.1.4.diff .

RUN wget https://www.php.net/distributions/php-8.1.4.tar.gz -qO - | tar xzf - \
    && cd ./php-8.1.4/ \
    && patch -p1 < ../php_8.1.4.diff \
    && ./configure \
        --prefix=/php/ \
        --enable-dba \
        --enable-fpm \
        --enable-gd \
        --enable-mysqlnd \
        --enable-zts \
        --enable-mbstring \
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
        --with-zlib \
    && sed -e 's/#define PHP_CAN_SUPPORT_PROC_OPEN 1//g' -i ./main/php_config.h \
    && sed -e 's/#define HAVE_FORK 1//g' -i ./main/php_config.h \
    && sed -e 's/#define HAVE_RFORK 1//g' -i ./main/php_config.h \
	&& make -j \
    && make install

COPY webserver .

RUN export CGO_CFLAGS_ALLOW=".*" \
    && export CGO_LDFLAGS_ALLOW=".*" \
    && go build -a

# second stage

FROM ubuntu:impish AS wp-builder

COPY ./build .

RUN apt-get update \
    && xargs -a packages.txt -r apt-get install -y \
    && rm -rf /var/lib/apt/lists/*

COPY wordpress .

RUN wget https://github.com/WordPress/WordPress/archive/refs/tags/5.9.3.zip -qO - | bsdtar -xf - \
    && patch -p1 -d WordPress-5.9.3/ < wordpress_5.9.3.diff \
    && mv WordPress-5.9.3/ wordpress/ \
    && mv wp-config.php wordpress/ \
    && wget https://www.akeeba.com/download/backupwp/7-6-4/akeebabackupwp-7-6-4-core-zip.raw -qO - | bsdtar -xf - -C wordpress/wp-content/plugins/ \
    && find wordpress/wp-content/themes/ -mindepth 1 -maxdepth 1 -type d -not -name twentytwentytwo -print0 | xargs -0 -I {} rm -r {} \
    && cd wordpress/ \
    && zip -rm ../app.zip .

# final container

FROM enclaive/gramine-os:latest

COPY ./packages.txt .

RUN apt-get update \
    && xargs -a packages.txt -r apt-get install -y \
    && rm -rf packages.txt /var/lib/apt/lists/*

# also works without this copy, saving 300mb
#COPY --from=builder /php/ /usr/
COPY --from=builder /era /usr/local/bin/
COPY --from=builder /phphttpd /app/

COPY --from=wp-builder /app.zip /app/

COPY ./webserver/tls/ /app/tls/
COPY ./php.manifest.template /app/
COPY ./entrypoint.sh /app/
COPY ./php.ini /php/lib/

#COPY ./backup.zip /app/app.zip

WORKDIR /app

RUN sed -e s/true/false/g -i /etc/sgx_default_qcnl.conf \
    && gramine-sgx-gen-private-key \
    && gramine-manifest -Dlog_level=error -Darch_libdir=/lib/x86_64-linux-gnu php.manifest.template php.manifest \
    && gramine-sgx-sign --manifest php.manifest --output php.manifest.sgx

VOLUME "/data"
EXPOSE 80 443
ENTRYPOINT ["/app/entrypoint.sh"]
