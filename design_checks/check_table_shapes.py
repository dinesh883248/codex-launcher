"""
MUST HAVE REQUIREMENTS:
- Verify every table includes a colgroup element with at least one col entry.
- Require each table to include a tbody section.
- Enforce that each td contains at most one child element.
- Keep tracking needed to flag missing table structure details.
- Read HTML directly from a provided path or stdin.
"""
# ----------------------------------
# Table scaffolding checks
# ----------------------------------
import sys
from html.parser import HTMLParser


class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.tables = []
        self.td_level = 0
        self.child_count = 0
        self.bad = ""

    def handle_starttag(self, tag, attrs):
        if self.bad:
            return
        if self.td_level:
            if self.td_level == 1:
                self.child_count += 1
                if self.child_count > 1:
                    self.bad = "td must not have more than one child element"
                    return
            self.td_level += 1
        if tag == "td":
            self.td_level = 1
            self.child_count = 0
        if tag == "table":
            self.tables.append([False, False, 0])
        elif tag == "colgroup" and self.tables:
            self.tables[-1][0] = True
        elif tag == "tbody" and self.tables:
            self.tables[-1][1] = True
        elif tag == "col" and self.tables:
            self.tables[-1][2] += 1

    def handle_endtag(self, tag):
        if self.bad:
            return
        if self.td_level:
            self.td_level -= 1
            if tag == "td":
                self.child_count = 0
        if tag == "table" and self.tables:
            cg, tb, cols = self.tables.pop()
            if not cg:
                self.bad = "table missing colgroup"
            elif not tb:
                self.bad = "table missing tbody"
            elif not cols:
                self.bad = "colgroup needs cols"


# ----------------------------------
# Report outcome
# ----------------------------------
src = sys.stdin.read() if len(sys.argv) == 1 else open(sys.argv[1]).read()
p = P()
p.feed(src)
if p.bad:
    print(p.bad)
    sys.exit(1)
print("table scaffolding satisfied")
