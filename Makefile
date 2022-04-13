ALL: pipedream


pipedream: php-8.1.4/libs/libphp.a .PHONY
	go build

php-8.1.4/libs/libphp.a:
	cd php-8.1.4 &&\
	./configure \
		--enable-static \
		--disable-shared \
		--enable-embed=static \
		--enable-dba \
        --enable-fpm \
        --enable-gd \
        --enable-mysqlnd \
        --with-password-argon2 \
        --with-bz2 \
        --with-curl \
        --with-imap \
        --with-imap-ssl \
        --with-kerberos \
        --with-mysqli=mysqlnd \
        --with-openssl \
        --with-pdo-mysql=mysqlnd \
        --with-pdo-sqlite \
        --with-readline \
        --with-zip \
        --with-zlib &&\
	make -jd


clean:
	cd php-8.1.4 && make clean



.PHONY:
