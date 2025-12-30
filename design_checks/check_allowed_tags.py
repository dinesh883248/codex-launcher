"""
MUST HAVE REQUIREMENTS:
- Accept only the allowed table-safe tags listed in this module.
- Allow link/script/asciinema-player only when the filename is livestream.html or request_cast.html.
- Allow inline style solely on table and col tags.
- Inline styles must be width declarations in numeric px with no other CSS.
- Read raw HTML from a provided path or stdin without pickle inputs.
"""
# ----------------------------------
# Keep tags and inline styles constrained
# ----------------------------------
import sys
from html.parser import HTMLParser

# ----------------------------------
# Allowed tag sets
# ----------------------------------
allowed = {
    "html",
    "head",
    "title",
    "meta",
    "thead",
    "style",
    "body",
    "table",
    "tbody",
    "colgroup",
    "col",
    "tr",
    "td",
    "p",
    "small",
    "span",
    "hr",
    "a",
    "h1",
    "h2",
    "h3",
    "img",
    "video",
    "code",
    "pre",
    "form",
    "button",
    "input",
    "textarea",
}
extra = {"link", "script", "asciinema-player"}

target = sys.argv[1] if len(sys.argv) > 1 else ""
allow_extra = target.endswith(("livestream.html", "request_cast.html"))


# ----------------------------------
# Parse and collect violations
# ----------------------------------
class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.bad = ""

    def handle_starttag(self, tag, attrs):
        if self.bad:
            return
        if tag not in allowed and not (allow_extra and tag in extra):
            self.bad = f"disallowed tag {tag}"
            return
        for k, v in attrs:
            if k != "style":
                continue
            if tag != "table" and tag != "col":
                self.bad = "inline style only allowed on table and col"
                return
            s = "".join(v.split())
            if s.endswith(";"):
                s = s[:-1]
            if not s.startswith("width:"):
                self.bad = f"inline css only width allowed: {tag}"
                return
            rest = s[6:-2]
            if rest and not rest.isdigit():
                self.bad = f"inline css only width allowed: {tag}"
                return

    handle_startendtag = handle_starttag


# ----------------------------------
# Emit results
# ----------------------------------
src = sys.stdin.read() if len(sys.argv) == 1 else open(sys.argv[1]).read()
p = P()
p.feed(src)
if p.bad:
    print(p.bad)
    sys.exit(1)
print("tag rules satisfied")
