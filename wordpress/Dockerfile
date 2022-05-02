#FROM enclaive/debug-gramine:latest
FROM enclaive/gramine-os:latest

RUN apt-get update &&\
    apt-get install -y sgx-aesm-service build-essential libssl-dev zlib1g zlib1g-dev wget \
    re2c libmariadb-dev libxml2-dev bison libsqlite3-dev libcurl4-openssl-dev libargon2-dev \
    libpng-dev libreadline-dev libz-dev zlib1g-dev libzip-dev libbz2-dev \
    libkrb5-dev vim less liblzma-dev netcat-openbsd curl mysql-client \
    zip &&\
    rm -rf /var/lib/apt/lists/* &&\
    wget https://go.dev/dl/go1.17.9.linux-amd64.tar.gz &&\
    tar -C /usr/local/ -xzf go1.17.9.linux-amd64.tar.gz &&\
    rm go1.17.9.linux-amd64.tar.gz &&\
    wget -P /bin https://github.com/edgelesssys/era/releases/latest/download/era &&\
    chmod +x /bin/era &&\
#build php &&\
    wget https://www.php.net/distributions/php-8.1.4.tar.gz &&\
    tar xvf php-8.1.4.tar.gz &&\
    rm php-8.1.4.tar.gz &&\
	cd php-8.1.4 &&\
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

COPY . /app/
WORKDIR /app

RUN \
    wget -q https://github.com/WordPress/WordPress/archive/refs/tags/5.9.3.zip -O WP_5.9.3.zip &&\
    unzip WP_5.9.3.zip &&\
    rm WP_5.9.3.zip && \
    patch -p1 -d WordPress-5.9.3/ < wordpress_5.9.3.diff &&\
    mv WordPress-5.9.3/ wordpress/ &&\
    mv wp-config.php wordpress/ &&\
    zip -rm app.zip wordpress/

RUN \
	export CGO_CFLAGS_ALLOW=".*" &&\
	export CGO_LDFLAGS_ALLOW=".*" &&\
	/usr/local/go/bin/go build -a &&\
# create sgxphp manifest &&\
    gramine-sgx-gen-private-key &&\
    gramine-manifest -Dlog_level=error -Darch_libdir=/lib/x86_64-linux-gnu php.manifest.template php.manifest &&\
    gramine-sgx-sign --manifest php.manifest --output php.manifest.sgx

ENTRYPOINT ["/app/entrypoint.sh"]

# ports
EXPOSE 80 443
