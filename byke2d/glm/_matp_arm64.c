#include <arm_neon.h>

struct vec3 {
  float x, y, z, w;
};

struct mat4 {
  struct vec3 columns[4];
};

void mul(struct mat4 *m, struct mat4 *o, struct mat4 *restrict out) {
  // float32x4_t c = vmulq_f32(a, b);
  // float32x4_t d = vfmaq_f32(a, b, c);
  float32x4_t mc0 = vld1q_f32((float *)&m->columns[0]);
  float32x4_t mc1 = vld1q_f32((float *)&m->columns[1]);
  float32x4_t mc2 = vld1q_f32((float *)&m->columns[2]);
  float32x4_t mc3 = vld1q_f32((float *)&m->columns[3]);

  for (int i = 0; i < 4; i++) {
    float32x4_t oX = vdupq_n_f32(o->columns[i].x);
    float32x4_t oY = vdupq_n_f32(o->columns[i].y);
    float32x4_t oZ = vdupq_n_f32(o->columns[i].z);
    float32x4_t oW = vdupq_n_f32(o->columns[i].w);

    float32x4_t resC = vmulq_f32(mc0, oX);
    resC = vfmaq_f32(resC, mc1, oY);
    resC = vfmaq_f32(resC, mc2, oZ);
    resC = vfmaq_f32(resC, mc3, oW);
    vst1q_f32((float *)&out->columns[i], resC);
  }
}
