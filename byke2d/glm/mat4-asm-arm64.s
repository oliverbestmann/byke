//go:build !nosimd && (arm64 && !goexperiment.simd)
#include "textflag.h"


TEXT ·mat4fMulAssign(SB), NOSPLIT, $0-16
    MOVD m+0(FP), R0
    MOVD o+8(FP), R1

    WORD $0xa9bf7bfd;    // stp x29, x30, [sp, #-16]!
    WORD $0x910003fd;    // mov x29, sp
    WORD $0x4d40c804;    // ld1r {v4.4s}, [x0]
    WORD $0x4c400838;    // ld4 {v24.4s-v27.4s}, [x1]
    WORD $0x2d42f81d;    // ldp s29, s30, [x0, #20]
    WORD $0x2d43d01f;    // ldp s31, s20, [x0, #28]
    WORD $0x2d41f015;    // ldp s21, s28, [x0, #12]
    WORD $0x2d40d817;    // ldp s23, s22, [x0, #4]
    WORD $0x4f9d933d;    // fmul v29.4s, v25.4s, v29.s[0]
    WORD $0xbd403c05;    // ldr s5, [x0, #60]
    WORD $0x4f9e933e;    // fmul v30.4s, v25.4s, v30.s[0]
    WORD $0x4f9f933f;    // fmul v31.4s, v25.4s, v31.s[0]
    WORD $0x4f9c933c;    // fmul v28.4s, v25.4s, v28.s[0]
    WORD $0x2d44c813;    // ldp s19, s18, [x0, #36]
    WORD $0x2d45c011;    // ldp s17, s16, [x0, #44]
    WORD $0x4f97131d;    // fmla v29.4s, v24.4s, v23.s[0]
    WORD $0x4f96131e;    // fmla v30.4s, v24.4s, v22.s[0]
    WORD $0x4f95131f;    // fmla v31.4s, v24.4s, v21.s[0]
    WORD $0x4e38cc9c;    // fmla v28.4s, v4.4s, v24.4s
    WORD $0x2d469807;    // ldp s7, s6, [x0, #52]
    WORD $0x4f93135d;    // fmla v29.4s, v26.4s, v19.s[0]
    WORD $0x4f92135e;    // fmla v30.4s, v26.4s, v18.s[0]
    WORD $0x4f91135f;    // fmla v31.4s, v26.4s, v17.s[0]
    WORD $0x4f94135c;    // fmla v28.4s, v26.4s, v20.s[0]
    WORD $0x4f87137d;    // fmla v29.4s, v27.4s, v7.s[0]
    WORD $0x4f86137e;    // fmla v30.4s, v27.4s, v6.s[0]
    WORD $0x4f85137f;    // fmla v31.4s, v27.4s, v5.s[0]
    WORD $0x4f90137c;    // fmla v28.4s, v27.4s, v16.s[0]
    WORD $0xa8c17bfd;    // ldp x29, x30, [sp], #16
    WORD $0x4c00081c;    // st4 {v28.4s-v31.4s}, [x0]
    WORD $0xd2800000;    // mov x0, #0x0                    // #0
    WORD $0xd2800001;    // mov x1, #0x0                    // #0

    RET


TEXT ·mat4fTranslate(SB), NOSPLIT, $0-20
    MOVD m+0(FP),  R0
    FMOVS x+8(FP), F0
    FMOVS y+12(FP), F1
    FMOVS z+16(FP), F2

    WORD $0xa9bf7bfd;    // stp x29, x30, [sp, #-16]!
    WORD $0x910003fd;    // mov x29, sp
    WORD $0xad407403;    // ldp q3, q29, [x0]
    WORD $0xad41781f;    // ldp q31, q30, [x0, #32]
    WORD $0x4f8193bd;    // fmul v29.4s, v29.4s, v1.s[0]
    WORD $0xa8c17bfd;    // ldp x29, x30, [sp], #16
    WORD $0x4f80107d;    // fmla v29.4s, v3.4s, v0.s[0]
    WORD $0x4f8213fd;    // fmla v29.4s, v31.4s, v2.s[0]
    WORD $0x4e3ed7be;    // fadd v30.4s, v29.4s, v30.4s
    WORD $0x3d800c1e;    // str q30, [x0, #48]
    WORD $0xd2800000;    // mov x0, #0x0                    // #0

    RET
    WORD $0xd503201f;    // nop
    WORD $0xd503201f;    // nop
    WORD $0xd503201f;    // nop


TEXT ·mat4fScale(SB), NOSPLIT, $0-20
    MOVD m+0(FP),  R0
    FMOVS x+8(FP), F0
    FMOVS y+12(FP), F1
    FMOVS z+16(FP), F2

    WORD $0xa9bf7bfd;    // stp x29, x30, [sp, #-16]!
    WORD $0x1e204043;    // fmov s3, s2
    WORD $0x910003fd;    // mov x29, sp
    WORD $0xad407c02;    // ldp q2, q31, [x0]
    WORD $0xa8c17bfd;    // ldp x29, x30, [sp], #16
    WORD $0x4f8193ff;    // fmul v31.4s, v31.4s, v1.s[0]
    WORD $0x3dc00801;    // ldr q1, [x0, #32]
    WORD $0x4f809042;    // fmul v2.4s, v2.4s, v0.s[0]
    WORD $0x4f839021;    // fmul v1.4s, v1.4s, v3.s[0]
    WORD $0xad007c02;    // stp q2, q31, [x0]
    WORD $0x3d800801;    // str q1, [x0, #32]
    WORD $0xd2800000;    // mov x0, #0x0                    // #0

    RET
