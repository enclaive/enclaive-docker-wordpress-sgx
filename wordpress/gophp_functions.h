#include <php/main/php.h>
#include <php/main/SAPI.h>
#include <php/main/php_main.h>
#include <php/main/php_variables.h>
#include <php/main/php_ini.h>
#include <php/Zend/zend_ini.h>

/* {{{ Fetch all HTTP request headers */
PHP_FUNCTION(apache_request_headers)
{
    array_init(return_value);

    fprintf(stderr, "[zend] CALL TO apache_request_headers not implemented\n");

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
