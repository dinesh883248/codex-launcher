"""
MUST HAVE REQUIREMENTS:
- Accept a target HTML path and prefer the virtualenv python if present.
- Run tidy first and use tidy's cleaned XHTML output for subsequent checks (including lxml strict).
- Execute every design check script using only an HTML path argument (no pickle handoff).
- Stop on the first failing script while printing its output.
- Print the original file checksum when all checks succeed.
"""
import hashlib
import os
import subprocess
import sys

orig = sys.argv[1]
target = orig
root = os.path.dirname(__file__)
venv_py = os.path.join(root, ".venv", "bin", "python3")
python = venv_py if os.path.exists(venv_py) else sys.executable
checks = [
    "design_checks/check_tidy_strict.py",
    "design_checks/check_lxml_strict.py",
    "design_checks/check_head_css.py",
    "design_checks/check_allowed_tags.py",
    "design_checks/check_attributes.py",
    "design_checks/check_body_tables.py",
    "design_checks/check_table_shapes.py",
    "design_checks/check_td_nbsp_pure.py",
    "design_checks/check_widths_via_dict_structure.py",
    "design_checks/check_forms.py",
    "design_checks/check_spacing_horizontal.py",
    "design_checks/check_spacing_vertical.py",
    "design_checks/check_spacing_vertical_between_tables.py",
    "design_checks/check_row_text_elements.py",
    "design_checks/check_px_units.py",
]
for rel in checks:
    path = os.path.join(root, rel)
    proc = subprocess.run([python, path, target], text=True, capture_output=True)
    if proc.returncode:
        print(proc.stdout + proc.stderr)
        sys.exit(proc.returncode)
    if rel.endswith("check_tidy_strict.py"):
        out = proc.stdout.strip().splitlines()
        if out:
            target = out[-1]

checksum = hashlib.sha256(open(orig, "rb").read()).hexdigest()
print(f"{orig} {checksum}")
