"""
MUST HAVE REQUIREMENTS:
- Enforce vertical spacing: adjacent content rows must have a spacer row between them unless both rows are stackable.
- Treat a spacer row as a row where every td has no child elements and only a single non-breaking space (U+00A0) as its text.
- Stackable tags: input, p, h1, h2, h3, pre, span, small, a, code.
- Always need gap tags: button, hr, img, video, plus a.link-button.
- Read HTML directly from a provided file path or stdin.

Intuition:
Think of each table as a vertical "stack" of rows. Some rows are real content; some rows are explicit gaps.
We classify rows into:
- SPACER_ROW: a deliberate blank gap row (all cells are exactly `&nbsp;`)
- CONTENT_ROW: any other row

Then we decide whether a CONTENT_ROW is stackable (can touch another CONTENT_ROW without a gap).
If either adjacent CONTENT_ROW is non-stackable, we require an explicit SPACER_ROW between them.
"""
# ----------------------------------
# Enforce <tr> spacer rows between vertical blocks
# ----------------------------------
import sys

from lxml import etree


# ----------------------------------
# Tags that "take space" and influence stackability decisions
# ----------------------------------
OCCUPY = {
    "input",
    "button",
    "img",
    "video",
    "hr",
    "p",
    "h1",
    "h2",
    "h3",
    "pre",
    "span",
    "small",
    "a",
    "code",
}
STACK = {"input", "p", "h1", "h2", "h3", "pre", "span", "small", "a", "code"}
GAP = {"button", "hr", "img", "video"}

# ----------------------------------
# Parse HTML input
# ----------------------------------
data = sys.stdin.buffer.read() if len(sys.argv) == 1 else open(sys.argv[1], "rb").read()
root = etree.HTML(data)

# ----------------------------------
# Per-table scan:
# - Spacer rows reset the "adjacent content" state.
# - Adjacent non-stackable content rows must have a spacer row between them.
# ----------------------------------
for ti, table in enumerate(root.iter("table")):
    prev = False
    prev_stack = True
    prev_i = -1
    prev_why = ""
    for i, tr in enumerate(table.findall("tbody/tr")):
        tds = tr.findall("td")

        # ----------------------------------
        # Identify a spacer row (all cells are true NBSP spacer cells)
        # ----------------------------------
        spacer = True
        for td in tds:
            if len(td):
                spacer = False
                break
            txt = "".join(td.itertext()).replace(" ", "").replace("\n", "").replace("\t", "").replace("\r", "")
            if txt != "\xa0":
                spacer = False
                break
        if spacer:
            prev = False
            continue

        # ----------------------------------
        # Decide whether this content row is stackable (allowed to touch)
        # ----------------------------------
        stackable = True
        why = ""
        for el in tr.iter():
            if el.tag == "a" and el.get("class") == "link-button":
                stackable = False
                why = "a.link-button"
                break
            if el.tag in GAP:
                stackable = False
                why = el.tag
                break
            if el.tag in OCCUPY and el.tag not in STACK:
                stackable = False
                why = el.tag
                break

        # ----------------------------------
        # Enforce vertical spacing between content rows
        # ----------------------------------
        if prev and not (prev_stack and stackable):
            tid = table.get("id")
            print(
                "missing spacer row between adjacent rows",
                "table",
                tid or ti,
                "between rows",
                prev_i,
                "and",
                i,
                "row1_reason",
                prev_why,
                "row2_reason",
                why,
            )
            sys.exit(1)
        prev = True
        prev_stack = stackable
        prev_i = i
        prev_why = why

print("vertical spacing satisfied")
