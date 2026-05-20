#!/usr/bin/env python3

import re
import sys

arm64_mul4f_mul_assign = """
TEXT ·mat4fMulAssign(SB), NOSPLIT, $0-16
    MOVD m+0(FP), R0
    MOVD o+8(FP), R1
"""

arm64_mul4f_scale_assign = """
TEXT ·mat4fScale(SB), NOSPLIT, $0-20
    MOVD m+0(FP),  R0
    FMOVS x+8(FP), F0
    FMOVS y+12(FP), F1
    FMOVS z+16(FP), F2
"""

arm64_mul4f_translate_assign = """
TEXT ·mat4fTranslate(SB), NOSPLIT, $0-20
    MOVD m+0(FP),  R0
    FMOVS x+8(FP), F0
    FMOVS y+12(FP), F1
    FMOVS z+16(FP), F2
"""

x64_mul4f_mul_assign = """
TEXT ·mat4fMulAssign(SB), NOSPLIT, $0-16
    MOVQ m+0(FP), DI
    MOVQ o+8(FP), SI
"""

x64_mul4f_scale_assign = """
TEXT ·mat4fScale(SB), NOSPLIT, $0-20
    MOVQ m+0(FP),  DI
    MOVSS x+8(FP), X0
    MOVSS y+12(FP), X1
    MOVSS z+16(FP), X2
"""

x64_mul4f_translate_assign = """
TEXT ·mat4fTranslate(SB), NOSPLIT, $0-20
    MOVD m+0(FP),  DI
    MOVSS x+8(FP), X0
    MOVSS y+12(FP), X1
    MOVSS z+16(FP), X2
"""

source = [line.strip() for line in sys.stdin]

is_x64 = any("elf64-x86-64" in line for line in source)
is_arm = any("elf64-littleaarch64" in line for line in source)

if is_x64:
    re_instr = re.compile(r"^\s*[a-f0-9]+:\s+([a-f0-9]{2}(?: [a-f0-9]{2})*)\s+(.*)")
    word = "BYTE"
    tags = "!nosimd && (amd64 && !goexperiment.simd)"

    stubs = {
        "mat4f_mul_assign": x64_mul4f_mul_assign,
        "mat4f_scale": x64_mul4f_scale_assign,
        "mat4f_translate": x64_mul4f_translate_assign,
    }

elif is_arm:
    re_instr = re.compile(r"^\s*[a-f0-9]+:\s+([a-f0-9]{8})\s+(.*)")
    word = "WORD"
    tags = "!nosimd && arm64"

    stubs = {
        "mat4f_mul_assign": arm64_mul4f_mul_assign,
        "mat4f_scale": arm64_mul4f_scale_assign,
        "mat4f_translate": arm64_mul4f_translate_assign,
    }

else:
    raise ValueError("unknown platform")

print(f"//go:build {tags}")
print('#include "textflag.h"')

for line in source:
    match = re.search(r"<([a-z0-9_]+)>", line)
    if match is not None:
        name = match.group(1)
        print()
        print(stubs[name])
        continue

    match = re_instr.search(line)
    if match is not None:
        instr, code = match.groups()
        if code == "ret":
            print()
            print("    RET")
            continue

        code = code.replace("\t", " ")
        instr = " ".join(f"{word} $0x{x};" for x in instr.split(" "))
        print(f"    {instr}    // {code}")
