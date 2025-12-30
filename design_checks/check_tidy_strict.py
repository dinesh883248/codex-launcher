"""
MUST HAVE REQUIREMENTS:
- Run tidy to rewrite input as XHTML and capture the cleaned output path.
- Accept stdin when no file path is provided by creating a temporary input file.
- Treat tidy errors (return code > 1 or stderr containing 'Error') as failures.
- Print the cleaned file path when tidy succeeds, even if warnings occurred.
- Allow custom tags only when the filename is livestream.html.
"""
# ----------------------------------
# Run tidy to emit normalized XHTML; warn-only ok
# ----------------------------------
import subprocess
import sys
from tempfile import NamedTemporaryFile

path = sys.argv[1] if len(sys.argv) > 1 else None
if not path:
    tmp_in = NamedTemporaryFile(delete=False, suffix=".html")
    tmp_in.write(sys.stdin.read().encode())
    tmp_in.flush()
    path = tmp_in.name

suffix = "livestream.html" if path.endswith("livestream.html") else ".html"
allow_extra = suffix != ".html"
tmp_out = NamedTemporaryFile(delete=False, suffix=suffix)
tmp_out.close()
cmd = [
    "tidy",
    "-numeric",
    "-asxhtml",
    "-utf8",
]
if allow_extra:
    cmd.extend(["--custom-tags", "blocklevel"])
cmd.extend(["-o", tmp_out.name, path])
proc = subprocess.run(cmd, capture_output=True, text=True)
msg = proc.stderr.strip()
if "Error" in msg or proc.returncode > 1:
    print(msg or "tidy reported issues")
    sys.exit(1)
print(tmp_out.name)
