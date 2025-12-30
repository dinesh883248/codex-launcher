"""
MUST HAVE REQUIREMENTS:
- If a td contains a non-breaking space (U+00A0) in its direct text, it must contain exactly one NBSP and nothing else (except ASCII whitespace).
- A td that is a pure NBSP gap cell must not contain any child elements.
- Read HTML directly from a provided file path or stdin.
"""
# ----------------------------------
# Reject td that mixes NBSP with other content
# ----------------------------------
import sys

from lxml import etree

data = sys.stdin.buffer.read() if len(sys.argv) == 1 else open(sys.argv[1], "rb").read()
root = etree.HTML(data)

for td in root.iter("td"):
    t = td.text
    if t and "\xa0" in t and (len(td) or t.strip(" \n\t\r") != "\xa0"):
        sys.exit("td nbsp must be sole content")

print("td nbsp purity satisfied")
