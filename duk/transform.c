/* transform.c */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "duktape.h"
#include "duk_print_alert.h"

/* For brevity assumes a maximum file length of 16kB. */
static void push_file_as_string(duk_context *ctx, const char *filename) {
    FILE *f;
    size_t len;
    char buf[16384];

    f = fopen(filename, "rb");
    if (f) {
        len = fread((void *) buf, 1, sizeof(buf), f);
        fclose(f);
        duk_push_lstring(ctx, (const char *) buf, (duk_size_t) len);
    } else {
        duk_push_undefined(ctx);
    }
}

int transform(const char *acc, const char** lines, int n_lines, char **out) {
    duk_context *ctx = NULL;
    int retval = 0;
    ctx = duk_create_heap_default();
    if (!ctx) {
        retval = 1;
        return retval;
    }
    duk_print_alert_init(ctx, 0 /*flags*/);
    push_file_as_string(ctx, "transform.js");
    if (duk_peval(ctx) != 0) {
        retval = 1;
        goto finished;
    }
    // ignore result
    duk_pop(ctx);
    duk_push_global_object(ctx);
    duk_get_prop_string(ctx, -1, "transform");
    duk_push_string(ctx, acc);
    duk_push_array(ctx);
    for (int i = 0; i < n_lines; i++) {
        duk_push_string(ctx, lines[i]);
        duk_put_prop_index(ctx, -2, i);
    }
    if (duk_pcall(ctx, 2 /* nargs */) != 0) {
        printf("Error: %s\n", duk_safe_to_string(ctx, -1));
        retval = 1;
        goto finished;
    }
    *out = strdup(duk_safe_to_string(ctx, -1));
    duk_pop(ctx);  /* pop result/error */
finished:
    duk_destroy_heap(ctx);
    return retval;
}

// int main(int argc, const char *argv[]) {
//     char *acc = "greetings";
//     char *lines[] = {"one", "stranger"};
//     char *out;
//     int err;
//     err = transform(acc, lines, 2, &out);
//     if (err != 0) {
//         printf("error while transform");
//         return err;
//     }
//     printf("out: %s\n", out);
//     free(out);
//     return 0;
// }
//     duk_context *ctx = NULL;
//     char line[4096];
//     size_t idx;
//     int ch;

//     (void) argc; (void) argv;

//     ctx = duk_create_heap_default();
//     if (!ctx) {
//         printf("Failed to create a Duktape heap.\n");
//         exit(1);
//     }

//     push_file_as_string(ctx, "process.js");
//     if (duk_peval(ctx) != 0) {
//         printf("Error: %s\n", duk_safe_to_string(ctx, -1));
//         goto finished;
//     }
//     duk_pop(ctx);  /* ignore result */

//     memset(line, 0, sizeof(line));
//     idx = 0;
//     for (;;) {
//         if (idx >= sizeof(line)) {
//             printf("Line too long\n");
//             exit(1);
//         }

//         ch = fgetc(stdin);
//         if (ch == 0x0a) {
//             line[idx++] = '\0';

//             duk_push_global_object(ctx);
//             duk_get_prop_string(ctx, -1 /*index*/, "processLine");
//             duk_push_string(ctx, line);
//             if (duk_pcall(ctx, 1 /*nargs*/) != 0) {
//                 printf("Error: %s\n", duk_safe_to_string(ctx, -1));
//             } else {
//                 printf("%s\n", duk_safe_to_string(ctx, -1));
//             }
//             duk_pop(ctx);  /* pop result/error */

//             idx = 0;
//         } else if (ch == EOF) {
//             break;
//         } else {
//             line[idx++] = (char) ch;
//         }
//     }

//  finished:
//     duk_destroy_heap(ctx);

//     exit(0);
// }
