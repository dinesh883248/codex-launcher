"""
MUST HAVE REQUIREMENTS:
- Scan inline style attributes for units that are not px.
- Treat any non-px unit match as a failure.
- Report success only when all styles use px units exclusively.
- Read HTML content directly from stdin or a provided file path.
"""
# ----------------------------------
# Enforce px-only units in inline styles
# ----------------------------------
import re
import sys
from html.parser import HTMLParser

unit_re = re.compile(r"\d+(?!px)[a-z%]+")


class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.bad = False

    def handle_starttag(self, tag, attrs):
        for name, value in attrs:
            if name == "style" and unit_re.search((value or "").lower()):
                self.bad = True


src = sys.stdin.read() if len(sys.argv) == 1 else open(sys.argv[1]).read()
p = P()
p.feed(src)
if p.bad:
    print("only px units allowed in inline styles")
    sys.exit(1)
print("px unit rules satisfied")
