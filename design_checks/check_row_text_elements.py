"""
MUST HAVE REQUIREMENTS:
- For every table except those with id=menu or id=breadcrumbs, allow at most one text element per row.
- Treat tags like p, span, small, headings, code, pre, and anchor as text elements.
- Exempt anchors with class link-button since they render as buttons.
- Read HTML directly from a provided file path or stdin.
"""
# ----------------------------------
# Enforce single text element per row (outside menu)
# ----------------------------------
import sys

from lxml import etree


text_tags = {
    "p",
    "span",
    "small",
    "a",
    "h1",
    "h2",
    "h3",
    "code",
    "pre",
}


data = sys.stdin.buffer.read() if len(sys.argv) == 1 else open(sys.argv[1], "rb").read()
root = etree.HTML(data)
for table in root.iter("table"):
    if table.get("id") in {"menu", "breadcrumbs"}:
        continue
    for tr in table.findall(".//tr"):
        count = 0
        for el in tr.iter():
            if el.tag not in text_tags:
                continue
            if el.tag == "a" and "link-button" in (el.get("class") or ""):
                continue
            count += 1
            if count > 1:
                print("rows outside menu can have only one text element")
                sys.exit(1)

print("row text element rules satisfied")
