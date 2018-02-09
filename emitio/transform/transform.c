/* transform.c */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "duktape.h"
#include "transform.h"

int transform(duk_context* ctx, const transform_param in, transform_param* out, char* err) {
    int retval = 0;
    duk_push_global_object(ctx);
    duk_get_prop_string(ctx, -1, "transform");
    duk_push_string(ctx, in.accumulator);
    duk_push_array(ctx);
    for (int i = 0; i < in.nlines; i++) {
        duk_push_string(ctx, in.lines[i]);
        duk_put_prop_index(ctx, -2, i);
    }
    if (duk_pcall(ctx, 2 /* nargs */) != 0) {
        printf("Error: %s\n", duk_safe_to_string(ctx, -1));
        retval = 1;
        goto finished;
    }
    duk_get_prop_string(ctx, -1, "length");
    duk_int_t l;
    l = duk_get_int(ctx, -1);
    duk_pop(ctx);
    if (l < 2) {
        printf("expected 2 items in result\n");
        retval = 1;
        goto finished;
    }
    duk_get_prop_index(ctx, -1, 0);
    if (!duk_is_string(ctx, -1)) {
        printf("expected first item to be string\n");
        retval = 1;
        duk_pop(ctx);
        goto finished;
    }
    out->accumulator = strdup(duk_safe_to_string(ctx, -1));
    duk_pop(ctx);
    duk_get_prop_index(ctx, -1, 1);
    if (!duk_is_array(ctx, -1)) {
        printf("expected second item to be array\n");
        retval = 1;
        duk_pop(ctx);
        goto finished;
    }
    duk_get_prop_string(ctx, -1, "length");
    l = duk_get_int(ctx, -1);
    duk_pop(ctx);
    out->nlines = (int)l;
    out->lines = malloc(out->nlines * sizeof(char*));
    for (int i = 0; i < l; i++) {
        duk_get_prop_index(ctx, -1, i);
        if (!duk_is_string(ctx, -1)) {
            printf("expected item in second result to be string\n");
            duk_pop(ctx);
            goto finished;
        }
        out->lines[i] = strdup(duk_safe_to_string(ctx, -1));
        duk_pop(ctx);
    }
    duk_pop(ctx);
finished:
    return retval;
}
