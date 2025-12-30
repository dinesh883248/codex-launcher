"""
MUST HAVE REQUIREMENTS:
- Enforce vertical spacing across adjacent top-level tables under body.
- Treat a spacer row as a row where every td has no child elements and only a single non-breaking space (U+00A0) as its text.
- Stackable tags: input, p, h1, h2, h3, pre, span, small, a, code.
- Always need gap tags: button, hr, img, video, plus a.link-button.
- Read HTML directly from a provided file path or stdin.
"""
# ----------------------------------
# Enforce spacer gaps between adjacent top-level tables under <body>
# ----------------------------------
import sys

from lxml import etree

data = sys.stdin.buffer.read() if len(sys.argv) == 1 else open(sys.argv[1], "rb").read()
root = etree.HTML(data)
body = root.find("body")

prev = False
prev_stack = True
prev_end_gap = False
for table in (body.findall("table") if body is not None else ()):
    if prev and table.get("id") == "breadcrumbs":
        prev = False
    seen = start_gap = end_gap = False
    first_stack = last_stack = True
    for tr in table.findall("tbody/tr"):
        spacer = True
        for td in tr:
            if len(td) or td.text != "\xa0":
                spacer = False
                break
        if spacer:
            if seen:
                end_gap = True
            else:
                start_gap = True
            continue

        stackable = not tr.xpath(".//a[@class='link-button']|.//button|.//hr|.//img|.//video")
        if not seen:
            first_stack = stackable
            seen = True
        last_stack = stackable
        end_gap = False

    if not seen:
        prev = False
        continue

    if prev and not (prev_end_gap or start_gap or (prev_stack and first_stack)):
        sys.exit("missing spacer row between adjacent tables")

    prev = True
    prev_stack = last_stack
    prev_end_gap = end_gap

print("vertical between-table spacing satisfied")
