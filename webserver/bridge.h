#include <stddef.h>

char *gophp_request_get_cookie    (void *ctx);
size_t gophp_request_read_post    (void *ctx, char *buf, size_t l);

size_t gophp_body_write (void *ctx, char*s, size_t l);

void gophp_response_headers_write (void *ctx, int response_code);
void gophp_response_headers_add   (void *ctx, char*v, size_t l);
void gophp_response_headers_del   (void *ctx, char*v, size_t l);
void gophp_response_headers_set   (void *ctx, char*v, size_t l);
void gophp_response_headers_clear (void *ctx);

void gophp_register_variables_each_php  (void *track_vars_array, char *k, char*v);
void gophp_register_variables_go        (void *ctx, void *track_vars_array);

int phpmain(
        void *ctx,
        char *script_path,
        char *request_method,
        char *request_uri,
        char *query_string,
        char *content_type,
        size_t content_length
);
