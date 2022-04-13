#include <sapi/embed/php_embed.h>
#include <stdio.h>

#include "gophp.h"



static int gophp_startup(sapi_module_struct *sapi_module) {
    if (php_module_startup(sapi_module, 0, 0) ==FAILURE) {
        return FAILURE;
    }

    fprintf(stderr, "gophp_startup all good\n");

    return SUCCESS;
}

static size_t gophp_read_post(char *buf, size_t count_bytes)
{
    return gophp_request_read_post(SG(server_context), buf, count_bytes);
}

static char* gophp_read_cookies(void)
{
    return gophp_request_get_cookie(SG(server_context));
}

static int gophp_deactivate(void)
{
    fprintf(stderr, "[zend] gophp_deactivate\n");

    if (SG(server_context) == 0) {
        return SUCCESS;
    }

    // TODO only run if not run before
    gophp_response_headers_write(SG(server_context), SG(sapi_headers).http_response_code);

    return SUCCESS;
}

static size_t gophp_ub_write(const char *str, size_t str_length) {

    const char *ptr = str;
    size_t remaining = str_length;
    size_t ret;

    while (remaining > 0) {
        ret = gophp_body_write(SG(server_context), (char*)ptr, remaining);
        if (!ret) {
            php_handle_aborted_connection();
        }
        ptr += ret;
        remaining -= ret;
    }

    return str_length;
}


static void gophp_flush(void *server_context) {
    if (fflush(stdout)==EOF) {
        php_handle_aborted_connection();
    }
}

static void gophp_log_message(const char *msg, int syslog_type_int) {
    fprintf(stderr, "[zend] %s\n", msg);
}



void gophp_register_variables_each_php(void *track_vars_array, char *k, char*v) {
    php_register_variable(k, v, (zval *)track_vars_array);
}

static void gophp_register_variables(zval *track_vars_array) {
    php_import_environment_variables(track_vars_array);
    gophp_register_variables_go(SG(server_context), track_vars_array);
}

static int gophp_send_headers_handler(sapi_headers_struct *sapi_headers)
{
    gophp_response_headers_write(SG(server_context), sapi_headers->http_response_code);
}

static int gophp_header_handler (sapi_header_struct *sapi_header, sapi_header_op_enum op, sapi_headers_struct *sapi_headers) {

	switch (op) {
		case SAPI_HEADER_REPLACE:
            gophp_response_headers_set(SG(server_context), sapi_header->header, sapi_header->header_len);
			return SAPI_HEADER_ADD;

		case SAPI_HEADER_ADD:
            gophp_response_headers_add(SG(server_context), sapi_header->header, sapi_header->header_len);
			return SAPI_HEADER_ADD;

		case SAPI_HEADER_DELETE:
            gophp_response_headers_del(SG(server_context), sapi_header->header, sapi_header->header_len);
			return 0;

		case SAPI_HEADER_DELETE_ALL:
            gophp_response_headers_clear(SG(server_context));
			return 0;

		case SAPI_HEADER_SET_STATUS:
            fprintf(stderr, "gophp_header_handler SAPI_HEADER_SET_STATUS not implemented\n");
			return 0;
	}
    return 0;
}


#include "gophp_functions.h"

static sapi_module_struct go2_sapi_module = {
    "embed",                        /* name */
    "PHP Embedded Library",         /* pretty name */

    gophp_startup,                  /* startup */
    php_module_shutdown_wrapper,    /* shutdown */

    NULL,                           /* activate */
    gophp_deactivate,               /* deactivate */

    gophp_ub_write,                 /* unbuffered write */
    gophp_flush,                    /* flush */
    NULL,                           /* get uid */
    NULL,                           /* getenv */

    php_error,                      /* error handler */

    gophp_header_handler,                           /* header handler */
    gophp_send_headers_handler,                     /* send headers handler */
    NULL,              /* send header handler */

    gophp_read_post,                /* read POST data */
    gophp_read_cookies,             /* read Cookies */

    gophp_register_variables,       /* register server variables */
    gophp_log_message,              /* Log message */
    NULL,                           /* Get request time */
    NULL,                           /* Child terminate */


    NULL, /* php_ini_path_override   */ \
    NULL, /* default_post_reader     */ \
    NULL, /* treat_data              */ \
    NULL, /* executable_location     */ \
    0,    /* php_ini_ignore          */ \
    0,    /* php_ini_ignore_cwd      */ \
    NULL, /* get_fd                  */ \
    NULL, /* force_http_10           */ \
    NULL, /* get_target_uid          */ \
    NULL, /* get_target_gid          */ \
    NULL, /* input_filter            */ \
    NULL, /* ini_defaults            */ \
    0,    /* phpinfo_as_text;        */ \
    NULL, /* ini_entries;            */ \
    ext_functions, /* additional_functions    */ \
    NULL  /* input_filter_init       */

};

static const char HARDCODED_INI[] =
    "html_errors=0\n"
    "register_argc_argv=1\n"
    "implicit_flush=1\n"
    "output_buffering=0\n"
    "max_execution_time=0\n"
    "max_input_time=-1\n\0";



int phpmain(
        void *ctx,
        char *script_path,
        char *request_method,
        char *request_uri,
        char *query_string,
        char *content_type,
        size_t content_length
) {

#ifdef ZTS
	php_tsrm_startup();
# ifdef PHP_WIN32
	ZEND_TSRMLS_CACHE_UPDATE();
# endif
#endif


	zend_signal_startup();
	sapi_startup(&go2_sapi_module);


    php_embed_module.ini_entries = malloc(sizeof(HARDCODED_INI));
    memcpy(php_embed_module.ini_entries, HARDCODED_INI, sizeof(HARDCODED_INI));


	if (go2_sapi_module.startup(&go2_sapi_module) != SUCCESS) {
        fprintf(stderr, "startup huh\n");
		return 1;
	}


    // => REQ

    SG(server_context) = ctx;


    SG(request_info).request_method  = request_method;
    SG(request_info).request_uri     = request_uri;
    SG(request_info).query_string    = query_string;
    SG(request_info).content_type    = content_type;
    SG(request_info).content_length  = content_length;

    fprintf(stderr, "[zend] content-length %d\n", content_length);
    fprintf(stderr, "[zend] request_uri  %s\n", request_uri);

    if (php_request_startup() == FAILURE) {
        php_module_shutdown();
        return FAILURE;
    }

    SG(headers_sent) = 1;
    SG(request_info).no_headers = 1;
    SG(sapi_headers).http_response_code = 200;



    FILE *fp = fopen(script_path, "rb");

    zend_file_handle zfd = {0};
    zend_stream_init_fp(&zfd, fp, script_path);
    zfd.primary_script = 1;




    //TODO zend_try


    int ret = php_execute_script(&zfd);

    zend_destroy_file_handle(&zfd);

    fprintf(stderr, "\nr:%d, memory_usage %d\n", ret, zend_memory_peak_usage(1));


    php_request_shutdown((void *) 0);


    // => REQ END?





    php_module_shutdown();
    sapi_shutdown();


#ifdef ZTS
    tsrm_shutdown();
#endif

    if (php_embed_module.ini_entries) {
        free(php_embed_module.ini_entries);
        php_embed_module.ini_entries = NULL;
    }



    fprintf(stderr, "[zend] phpmain done\n");

    return 0;
}
