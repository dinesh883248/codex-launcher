"""
MUST HAVE REQUIREMENTS:
- Fail when a body tag is missing from the document.
- Require every direct body child element to be a table.
- Ensure the body contains at least one table child element.
- Read HTML content directly from a file path or stdin.
"""
# ----------------------------------
# Body should be table-only
# ----------------------------------
import sys
from html.parser import HTMLParser


# ----------------------------------
# Track direct children under body
# ----------------------------------
class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.in_body = False
        self.body_depth = 0
        self.body_seen = False
        self.child_count = 0
        self.bad = ""

    def handle_starttag(self, tag, attrs):
        if tag == "body":
            self.in_body = True
            self.body_seen = True
            self.body_depth = 0
            return
        if not self.in_body:
            return
        if self.body_depth == 0:
            self.child_count += 1
            if tag != "table":
                if not self.bad:
                    self.bad = "body children must be table elements"
        self.body_depth += 1

    def handle_endtag(self, tag):
        if tag == "body":
            self.in_body = False
            self.body_depth = 0
            return
        if self.in_body and self.body_depth:
            self.body_depth -= 1


# ----------------------------------
# Validate order
# ----------------------------------
src = sys.stdin.read() if len(sys.argv) == 1 else open(sys.argv[1]).read()
p = P()
p.feed(src)
if not p.body_seen:
    print("missing body")
    sys.exit(1)
if not p.child_count:
    print("body needs table children")
    sys.exit(1)
if p.bad:
    print(p.bad)
    sys.exit(1)
print("body layout satisfied")
