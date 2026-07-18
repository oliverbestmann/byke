#include "radixsort.h"

#include <string.h>

/* Convert float bit-pattern into sortable uint32_t. */
static inline uint32_t float_to_sortable_u32(float f) {
  union {
    float f;
    uint32_t u;
  } v = {f};
  uint32_t x = v.u;

  uint32_t mask = ((int32_t)x >> 31) | 0x80000000u;
  return x ^ mask;
}

void radixsort_c(value_t *restrict data, value_t *restrict scratch,
                 uint32_t n) {
  value_t *restrict src = data;
  value_t *restrict dst = scratch;

  const uint32_t RADIX = 256;

  for (int pass = 0; pass < 4; ++pass) {
    uint32_t count[RADIX] = {};

    const int shift = pass * 8;

    /* Histogram */
    for (uint32_t i = 0; i < n; ++i) {
      uint32_t k = float_to_sortable_u32(src[i].key);
      count[(k >> shift) & 0xFF]++;
    }

    /* Prefix sum */
    uint32_t sum = 0;
    for (uint32_t i = 0; i < RADIX; ++i) {
      uint32_t c = count[i];
      count[i] = sum;
      sum += c;
    }

    /* Scatter */
    for (uint32_t i = 0; i < n; ++i) {
      value_t v = src[i];
      uint32_t k = float_to_sortable_u32(v.key);

      uint32_t idx = (k >> shift) & 0xFF;
      dst[count[idx]++] = v;
    }

    /* swap buffers */
    value_t *tmp = src;
    src = dst;
    dst = tmp;
  }
}
