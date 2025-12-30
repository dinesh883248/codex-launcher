"""
MUST HAVE REQUIREMENTS:
- Read Python source from files or stdin.
- Compile in exec mode and count bytecode instructions across all nested code objects.
- Print the total instruction count as an integer.
"""
# ----------------------------------
# Read source and compile once
# ----------------------------------
import dis
import fileinput
import types

src = "".join(fileinput.input())
root = compile(src, "<input>", "exec")

# ----------------------------------
# Walk nested code objects and count instructions
# ----------------------------------
stack = [root]
count = 0
while stack:
    co = stack.pop()
    for _ in dis.get_instructions(co):
        count += 1
    for const in co.co_consts:
        if isinstance(const, types.CodeType):
            stack.append(const)

print(count)
