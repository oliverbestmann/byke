#include <immintrin.h>

struct vec3 {
    float x, y, z, w;
};

struct mat4 {
    struct vec3 columns[4];
};

void mul(struct mat4 *m,
         struct mat4 *o,
         struct mat4 *restrict out)
{
    __m128 mc0 = _mm_loadu_ps((float *)&m->columns[0]);
    __m128 mc1 = _mm_loadu_ps((float *)&m->columns[1]);
    __m128 mc2 = _mm_loadu_ps((float *)&m->columns[2]);
    __m128 mc3 = _mm_loadu_ps((float *)&m->columns[3]);

    for (int i = 0; i < 4; i++) {
        __m128 oX = _mm_set1_ps(o->columns[i].x);
        __m128 oY = _mm_set1_ps(o->columns[i].y);
        __m128 oZ = _mm_set1_ps(o->columns[i].z);
        __m128 oW = _mm_set1_ps(o->columns[i].w);

        __m128 res = _mm_mul_ps(mc0, oX);

        res = _mm_fmadd_ps(mc1, oY, res);
        res = _mm_fmadd_ps(mc2, oZ, res);
        res = _mm_fmadd_ps(mc3, oW, res);

        _mm_storeu_ps((float *)&out->columns[i], res);
    }
}
