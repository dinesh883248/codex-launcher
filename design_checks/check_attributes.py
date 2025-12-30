"""
MUST HAVE REQUIREMENTS:
- Allow only the explicit attribute set per tag defined in allowed_attrs.
- Permit an id attribute solely on table elements.
- When an anchor has a class, it must be exactly link-button.
- Read HTML directly from a path or stdin without pickle staging.
"""
# ----------------------------------
# Restrict attributes on most tags
# ----------------------------------
import sys
from html.parser import HTMLParser

allowed_attrs = {
    "html": {"xmlns"},
    "form": {"method", "action"},
    "table": {"style", "id"},
    "col": {"style"},
    "button": {"type", "disabled"},
    "input": {"name", "type", "value", "placeholder", "disabled", "autofocus"},
    "meta": {"http-equiv", "content", "charset", "name"},
    "textarea": {"name", "hidden", "rows", "disabled"},
    "a": {"href", "class"},
    "img": {"src"},
    "iframe": {"src", "style"},
}


class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.bad = ""

    def handle_starttag(self, tag, attrs):
        if self.bad:
            return
        names = allowed_attrs[tag] if tag in allowed_attrs else ()
        for name, _ in attrs:
            if name not in names:
                self.bad = f"{tag} cannot have attribute {name}"
                return
        if tag == "a":
            for name, val in attrs:
                if name == "class" and val != "link-button":
                    self.bad = "a class must be link-button"
                    return

    handle_startendtag = handle_starttag


src = sys.stdin.read() if len(sys.argv) == 1 else open(sys.argv[1]).read()
p = P()
p.feed(src)
if p.bad:
    print(p.bad)
    sys.exit(1)
print("attribute rules satisfied")
