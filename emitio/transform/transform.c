/* transform.c */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "duktape.h"
#include "transform.h"

int transform(duk_context* ctx, const transform_param in, transform_param* out, char* err) {
    duk_push_global_object(ctx); // [global]
    if (duk_is_error(ctx, -1)) {
        sprintf(err, "%s", duk_safe_to_string(ctx, -1));
        duk_set_top(ctx, 0);
        return 1;
    }
    duk_get_prop_string(ctx, -1, "transform"); // [global, transform]
    if (duk_is_error(ctx, -1)) {
        sprintf(err, "%s", duk_safe_to_string(ctx, -1));
        duk_set_top(ctx, 0);
        return 1;
    }    
    duk_push_string(ctx, in.accumulator);
    duk_push_array(ctx);
    for (int i = 0; i < in.nlines; i++) {
        duk_push_string(ctx, in.lines[i]);
        duk_put_prop_index(ctx, -2, i);
    }
    if (duk_pcall(ctx, 2 /* nargs */) != 0) {
        sprintf(err, "%s", duk_safe_to_string(ctx, -1));
        duk_set_top(ctx, 0);
        return 1;
    }
    duk_get_prop_string(ctx, -1, "length");
    if (duk_is_error(ctx, -1)) {
        sprintf(err, "%s", duk_safe_to_string(ctx, -1));
        duk_set_top(ctx, 0);
        return 1;
    }    
    duk_int_t l;
    l = duk_get_int(ctx, -1);
    duk_pop(ctx);
    if (l < 2) {
        sprintf(err, "expected 2 items in result but got %d", l);
        duk_set_top(ctx, 0);
        return 1;
    }
    duk_get_prop_index(ctx, -1, 0);
    if (!duk_is_string(ctx, -1)) {
        sprintf(err, "expected first item to be string");
        duk_set_top(ctx, 0);
        return 1;
    }
    out->accumulator = strdup(duk_safe_to_string(ctx, -1));
    duk_pop(ctx);
    duk_get_prop_index(ctx, -1, 1);
    if (!duk_is_array(ctx, -1)) {
        sprintf(err, "expected second item to be array");
        duk_set_top(ctx, 0);
        return 1;
    }
    duk_get_prop_string(ctx, -1, "length");
    l = duk_get_int(ctx, -1);
    duk_pop(ctx);
    out->nlines = (int)l;
    out->lines = malloc(out->nlines * sizeof(char*));
    for (int i = 0; i < l; i++) {
        duk_get_prop_index(ctx, -1, i);
        if (!duk_is_string(ctx, -1)) {
            sprintf(err, "expected item idx=%d in second result to be string", i);
            duk_set_top(ctx, 0);
            return 1;
        }
        out->lines[i] = strdup(duk_safe_to_string(ctx, -1));
        duk_pop(ctx);
    }
    duk_set_top(ctx, 0);
    return 0;
}
