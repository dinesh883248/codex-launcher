"""
MUST HAVE REQUIREMENTS:
- Collect CSS from style blocks that appear inside head.
- Normalize gathered CSS (strip whitespace and CDATA markers) before comparison.
- Fail when normalized CSS differs from immutable.css and pass otherwise.
- Load HTML directly from a provided path or stdin without pickle helpers.
"""
# ----------------------------------
# Ensure head CSS matches immutable.css
# ----------------------------------
import os
import sys
from html.parser import HTMLParser


# ----------------------------------
# Gather style text inside head
# ----------------------------------
class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.head = False
        self.in_style = False
        self.css = []

    def handle_starttag(self, tag, attrs):
        if tag == "head":
            self.head = True
        if tag == "style" and self.head:
            self.in_style = True

    def handle_endtag(self, tag):
        if tag == "head":
            self.head = False
        if tag == "style":
            self.in_style = False

    def handle_data(self, data):
        if self.in_style:
            self.css.append(data)


# ----------------------------------
# Run the comparison
# ----------------------------------
src = sys.stdin.read() if len(sys.argv) == 1 else open(sys.argv[1]).read()
p = P()
p.feed(src)
path = os.path.join(os.path.dirname(__file__), "immutable.css")
target = "".join(open(path).read().split())
found = "".join("".join(p.css).replace("<![CDATA[", "").replace("]]>", "").split())
if target != found:
    print("head css must match immutable.css")
    sys.exit(1)
print("head css ok")
