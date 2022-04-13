#include <sapi/embed/php_embed.h>

/* {{{ Fetch all HTTP request headers */
PHP_FUNCTION(apache_request_headers)
{
    array_init(return_value);

    fprintf(stderr, "\n!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n!!!!! CALL TO apache_request_headers !!!!!!!!!!\n");

    //foreach
// add_assoc_string(return_value, key, val);
}

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_apache_request_headers, 0, 0, IS_ARRAY, 0)
ZEND_END_ARG_INFO()

#define arginfo_getallheaders arginfo_apache_request_headers

ZEND_FUNCTION(apache_request_headers);

static const zend_function_entry ext_functions[] = {
    ZEND_FE(apache_request_headers, arginfo_apache_request_headers)
    ZEND_FALIAS(getallheaders, apache_request_headers, arginfo_getallheaders)
    ZEND_FE_END
};
