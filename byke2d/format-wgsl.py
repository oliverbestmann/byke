#!/usr/bin/env python3

import argparse
import re
import sys
from pathlib import Path

OPENERS = "([{"
CLOSERS = ")]}"

INDENT = 4


def bracket_delta(code: str) -> tuple[int, int]:
    """Number of brackets opened (may be negative) by this line."""

    # we dont care what happens within a comment
    code = re.sub(r"//.*", "", code)

    # ignore any kind of whitespace within the line
    code = re.sub(r"\s+", "", code)

    # remove closers from the start of the line, i.e. "} else {"
    code_rest = code.lstrip(CLOSERS)

    # dedent to apply early
    dedent_first = len(code) - len(code_rest)

    # delta to apply after the line
    openers = sum(code_rest.count(c) for c in OPENERS)
    closers = sum(code_rest.count(c) for c in CLOSERS)
    delta = openers - closers

    return dedent_first, delta


def format_source(source: str) -> str:
    depth = 0

    result = []

    for raw in source.splitlines():
        line = raw.strip()

        if not line:
            result.append("")
            continue

        line = line.strip()

        # macros always live at column zero and do not influence the indentation
        if line.startswith("#"):
            result.append(line)
            continue

        dedent_first, delta = bracket_delta(line)

        depth -= dedent_first
        assert depth >= 0, "indentation is negative near: " + line

        result.append(((depth * INDENT) * " ") + line)

        depth += delta
        assert depth >= 0, "indentation is negative near: " + line

    # drop trailing empty lines, keep exactly one final newline
    while result and not result[-1]:
        result.pop()

    assert depth == 0, "indentation is not zero at eof"

    return "\n".join(result) + "\n"


def wgsl_files(paths: list[Path]):
    for path in paths:
        if path.is_dir():
            yield from sorted(path.rglob("*.wgsl"))
        elif path.suffix == ".wgsl":
            yield path
        else:
            print(f"skipping {path}: not a wgsl file", file=sys.stderr)


def main() -> int:
    parser = argparse.ArgumentParser(description="Format wgsl shader files in place.")
    parser.add_argument(
        "paths",
        nargs="*",
        default=["."],
        type=Path,
        help="directories or files to format (default: current directory)",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="do not write, exit non zero if a file would change",
    )
    args = parser.parse_args()

    changed = []

    for file in wgsl_files([Path(p) for p in args.paths]):
        source = file.read_text(encoding="utf8")

        formatted = format_source(source)
        if formatted == source:
            continue

        changed.append(file)
        if not args.check:
            file.write_text(formatted, encoding="utf8")

    for file in changed:
        verb = "would reformat" if args.check else "formatted"
        print(f"{verb} {file}")

    return 1 if (args.check and changed) else 0


if __name__ == "__main__":
    sys.exit(main())
