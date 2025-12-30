"""
MUST HAVE REQUIREMENTS:
- Build per-table dictionaries before validating widths; keep the dict approach.
- Use width parsing to pull integers from style attributes for tables and cols.
- Reject inline styles that set width:0px on any non-table element.
- Require every table to declare a width style and enforce 380px for tables that are direct body children.
- Require table width to match its assigned column width (380px for body tables).
- Demand positive colgroup widths that sum exactly to the table width.
- Enforce that every row's cell count equals the column count from colgroup.
- Parse HTML directly from a provided path or stdin without pickle helpers.

Intuition:
Treat each table as a little width contract:
- The table declares its own width.
- The colgroup declares widths that must sum to the table width.
- Nested tables must match the column width of the cell they sit in.
"""

# ----------------------------------
# Width rules via per-table dict structure
# ----------------------------------
import sys

from lxml import etree


# ----------------------------------
# Parse width digits from a style string (returns int or None)
# ----------------------------------
def pw(style):
    if not style:
        return None
    digits = ""
    for ch in style:
        if "0" <= ch <= "9":
            digits += ch
    return int(digits) if digits else None


data = sys.stdin.buffer.read() if len(sys.argv) == 1 else open(sys.argv[1], "rb").read()
root = etree.HTML(data)

# ----------------------------------
# Reject width:0px on non-table elements (carry-over from deleted check_table_widths.py)
# ----------------------------------
for el in root.iter():
    if el.tag == "table":
        continue
    s = el.get("style")
    if s and "width:0px" in s.replace(" ", "").lower():
        print("inline style must not use width: 0px")
        sys.exit(1)

# ----------------------------------
# Build table dictionaries first (required approach)
# ----------------------------------
tables = []
for el in root.iter("table"):
    style = el.get("style")
    w = pw(style)
    if w is None:
        print("table needs width style")
        sys.exit(1)

    parent = el.getparent()
    is_body = parent is not None and parent.tag == "body"
    assigned = 380
    if not is_body:
        while parent is not None and parent.tag != "td":
            parent = parent.getparent()
        if parent is None:
            print("table width must match assigned column width")
            sys.exit(1)
        tr = parent.getparent()
        outer = tr.getparent().getparent()
        idx = 0
        for i, td in enumerate(tr.findall("td")):
            if td is parent:
                idx = i
                break
        cols = outer.findall("colgroup/col")
        assigned = pw(cols[idx].get("style") if idx < len(cols) else None)
        if assigned is None:
            print("col needs width style")
            sys.exit(1)

    widths = []
    for col in el.findall("colgroup/col"):
        cw = pw(col.get("style"))
        if cw is None:
            print("col needs width style")
            sys.exit(1)
        if not cw:
            print("col width must be positive")
            sys.exit(1)
        widths.append(cw)

    rows = []
    for tr in el.findall("tbody/tr"):
        rows.append(len(tr.findall("td")))

    tables.append(
        {
            "width": w,
            "colgroup_widths": widths,
            "col_count": len(widths),
            "rows": rows,
            "is_body_child": is_body,
            "assigned_col_width": assigned,
        }
    )

# ----------------------------------
# Validate tables
# ----------------------------------
for t in tables:
    if t["is_body_child"] and t["width"] != 380:
        print("tables under body must be width 380px")
        sys.exit(1)
    if t["width"] != t["assigned_col_width"]:
        print("table width must match assigned column width")
        sys.exit(1)
    if sum(t["colgroup_widths"]) != t["width"]:
        print("table width must equal sum of colgroup widths")
        sys.exit(1)
    for cells in t["rows"]:
        if cells != t["col_count"]:
            print("row cell count must match colgroup column count")
            sys.exit(1)

print("nested widths satisfied")
