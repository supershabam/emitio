#include "duktape.h"

typedef char* cstring;

typedef struct transform_in transform_in;

struct transform_in {
    char* accumulator;
    cstring* lines;
    int nlines;
};

int transform(duk_context*, const transform_in, transform_in*);
