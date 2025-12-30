"""
MUST HAVE REQUIREMENTS:
- Parse the provided HTML as strict XML via lxml with recover disabled.
- Exit with an error message when parsing fails instead of silently fixing markup.
- Print success acknowledgement on valid input.
"""
# ----------------------------------
# Parse as strict XHTML with lxml (no recovery)
# ----------------------------------
import sys

from lxml import etree

path = sys.argv[1] if len(sys.argv) > 1 else None
data = sys.stdin.read() if not path else open(path).read()
try:
    etree.fromstring(data.encode(), parser=etree.XMLParser(recover=False))
except Exception as exc:
    print(f"xml parse failed: {exc}")
    sys.exit(1)
print("lxml strict ok")
