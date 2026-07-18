#include <stdint.h>

typedef struct value {
  float key;
  uint32_t index;
} value_t;

void radixsort_c(value_t *restrict data, value_t *restrict scratch, uint32_t n);
