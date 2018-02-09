typedef char* cstring;

typedef struct transform_in transform_in;

struct transform_in {
    char* accumulator;
    cstring* lines;
    int nlines;
};

int transform(const transform_in, transform_in*);