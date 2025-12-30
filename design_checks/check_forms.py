"""
MUST HAVE REQUIREMENTS:
- Treat a form as valid only when its parent tag is td.
- Require each form to contain a table before the form closes.
- Preserve the failure message when a form breaks either rule.
- Read HTML straight from a file path or stdin input.
"""
# ----------------------------------
# Forms must sit in cells and wrap tables
# ----------------------------------
import sys
from html.parser import HTMLParser


# ----------------------------------
# Track form context
# ----------------------------------
class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.stack = []
        self.forms = []
        self.bad = ""

    def handle_starttag(self, tag, attrs):
        if self.bad:
            return
        parent = self.stack[-1] if self.stack else ""
        if tag == "form":
            self.forms.append([parent == "td", False])
        if tag == "table" and self.forms:
            self.forms[-1][1] = True
        self.stack.append(tag)

    def handle_endtag(self, tag):
        if self.bad:
            return
        if self.stack:
            self.stack.pop()
        if tag == "form" and self.forms:
            ok_td, ok_table = self.forms.pop()
            if not ok_td or not ok_table:
                self.bad = "form must live inside td and wrap a table"


# ----------------------------------
# Emit verdict
# ----------------------------------
src = sys.stdin.read() if len(sys.argv) == 1 else open(sys.argv[1]).read()
p = P()
p.feed(src)
if p.bad:
    print(p.bad)
    sys.exit(1)
print("form rules satisfied")
