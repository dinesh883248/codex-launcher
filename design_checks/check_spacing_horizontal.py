"""
MUST HAVE REQUIREMENTS:
- Enforce horizontal spacing: between any 2 space-occupying cells in the same row, require a <td>&nbsp;</td> spacer cell.
- Treat a spacer cell as a td with no child elements and only a single non-breaking space (U+00A0) as its text.
- Treat "space-occupying" tags as: input, button, img, video, hr, p, h1, h2, h3, pre, span, small, a, code.
- Read HTML directly from a provided file path or stdin.

Intuition:
This check treats each table row like a simple "token stream" of cells:
CONTENT cells (contain a space-occupying element) must never touch each other directly.
If two CONTENT cells appear in the same row, there must be an explicit spacer cell between them:
`<td>&nbsp;</td>`. Empty `<td></td>` does not count because it does not create visible gap.
"""
# ----------------------------------
# Enforce <td>&nbsp;</td> between horizontal elements
# ----------------------------------
import sys

from lxml import etree


# ----------------------------------
# Tags that are considered "visually present" and therefore need spacing
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

# ----------------------------------
# Parse HTML input
# ----------------------------------
data = sys.stdin.buffer.read() if len(sys.argv) == 1 else open(sys.argv[1], "rb").read()
root = etree.HTML(data)

# ----------------------------------
# Row scan:
# - Once we see a CONTENT cell, we require a spacer before the next CONTENT cell.
# - The requirement is cleared only by a true spacer cell (<td>&nbsp;</td>).
# ----------------------------------
for ti, table in enumerate(root.iter("table")):
    tid = table.get("id")
    for ri, tr in enumerate(table.findall("tbody/tr")):
        need = False
        prev = -1
        for ci, td in enumerate(tr.findall("td")):
            # ----------------------------------
            # Identify a spacer cell (no children, text is exactly NBSP)
            # ----------------------------------
            sp = False
            if not len(td):
                txt = "".join(td.itertext()).replace(" ", "").replace("\n", "").replace("\t", "").replace("\r", "")
                sp = txt == "\xa0"

            # ----------------------------------
            # Identify whether this cell contains a space-occupying element
            # ----------------------------------
            occ = False
            for el in td.iter():
                if el.tag in OCCUPY:
                    occ = True
                    break

            # ----------------------------------
            # Enforce spacing between CONTENT cells
            # ----------------------------------
            if occ:
                if need:
                    print(
                        "missing <td>&nbsp;</td> spacer between elements",
                        "table",
                        tid or ti,
                        "row",
                        ri,
                        "between td",
                        prev,
                        "and",
                        ci,
                    )
                    sys.exit(1)
                need = True
                prev = ci
            elif need and sp:
                need = False

print("horizontal spacing satisfied")
