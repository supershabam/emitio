#include "duktape.h"

typedef char* cstring;

typedef struct transform_param transform_param;

struct transform_param {
    char* accumulator;
    cstring* lines;
    int nlines;
};

int transform(duk_context*, const transform_param, transform_param*, char*);
