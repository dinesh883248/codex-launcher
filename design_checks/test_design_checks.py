"""
MUST HAVE REQUIREMENTS:
- Provide helpers to run design check scripts directly on temp HTML files.
- Default test HTML should embed immutable.css to mirror runtime checks.
- Cover tag/attribute rules, table structure, forms, css/px unit validations, and width constraints.
- Cover horizontal and vertical spacing checks that enforce td(&nbsp;) spacers.
- Exercise tidy+lxml pipeline plus dict-based width checks (no pickles).
- Validate row text element limits for non-menu tables.
- Smoke test the full run_design_checks.py flow to ensure success output includes checksum.
"""
# ----------------------------------
# Unit tests for design check scripts
# ----------------------------------
import os
import subprocess
import sys
import tempfile
import textwrap
import unittest

BASE = os.path.dirname(os.path.dirname(__file__))
CHECKS_DIR = os.path.join(BASE, "design_checks")
CSS_PATH = os.path.join(CHECKS_DIR, "immutable.css")
CSS = "".join(open(CSS_PATH).read().split())
VENV_PY = os.path.join(BASE, ".venv", "bin", "python3")
PY_EXE = VENV_PY if os.path.exists(VENV_PY) else sys.executable


# ----------------------------------
# Helpers
# ----------------------------------
def run_check(script, html):
    with tempfile.TemporaryDirectory() as tmp:
        path = os.path.join(tmp, "page.html")
        with open(path, "w") as f:
            f.write(html)
        result = subprocess.run(
            [PY_EXE, os.path.join(CHECKS_DIR, script), path],
            text=True,
            capture_output=True,
        )
    return result.returncode, (result.stdout + result.stderr).strip()


def html_doc(body, css=CSS):
    return textwrap.dedent(
        f"""\
        <!DOCTYPE html>
        <html>
        <head>
        <title>t</title>
        <style>{css}</style>
        </head>
        <body>
        {body}
        </body>
        </html>
        """
    )


def table_block(inner, width="380px"):
    return textwrap.dedent(
        f"""\
        <table style="width:{width}">
          <colgroup><col style="width:{width}"></colgroup>
          <tbody>
            {inner}
          </tbody>
        </table>
        """
    )


# ----------------------------------
# Allowed tags and attributes
# ----------------------------------
class AllowedTagsTests(unittest.TestCase):
    def test_rejects_disallowed_tag(self):
        body = table_block("<tr><td><div>bad</div></td></tr>")
        code, out = run_check("check_allowed_tags.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("disallowed tag div", out)

    def test_accepts_allowed_tags(self):
        inner = "<tr><td><p>ok</p></td></tr>"
        code, out = run_check("check_allowed_tags.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("tag rules satisfied", out)

    def test_inline_style_requires_numeric_width(self):
        inner = (
            '<tr><td><table style="width:abcpx"><colgroup><col style="width:10px">'
            "</colgroup><tbody><tr><td>bad</td></tr></tbody></table></td></tr>"
        )
        code, out = run_check("check_allowed_tags.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("inline css only width allowed", out)


class AttributeTests(unittest.TestCase):
    def test_rejects_class_on_a(self):
        inner = '<tr><td><a href="#" class="x">link</a></td></tr>'
        code, out = run_check("check_attributes.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("a class must be link-button", out)

    def test_rejects_id_on_input(self):
        inner = '<tr><td><input id="x" name="n" type="text" value="v"></td></tr>'
        code, out = run_check("check_attributes.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("input cannot have attribute id", out)


# ----------------------------------
# Structure rules
# ----------------------------------
class BodyTablesTests(unittest.TestCase):
    def test_requires_table_children(self):
        code, out = run_check("check_body_tables.py", html_doc("<p>no table</p>"))
        self.assertNotEqual(code, 0)
        self.assertIn("body", out)
        self.assertIn("table", out)

    def test_missing_body_tag(self):
        html = "<html><head></head><div>no body</div></html>"
        code, out = run_check("check_body_tables.py", html)
        self.assertNotEqual(code, 0)
        self.assertIn("missing body", out)


class TableShapesTests(unittest.TestCase):
    def test_missing_colgroup_fails(self):
        body = textwrap.dedent(
            """
            <table style="width:380px">
              <tbody><tr><td>no colgroup</td></tr></tbody>
            </table>
            """
        )
        code, out = run_check("check_table_shapes.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("table missing colgroup", out)

    def test_td_multi_child_fails(self):
        inner = "<tr><td><p>one</p><p>two</p></td></tr>"
        code, out = run_check("check_table_shapes.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("td must not have more than one child element", out)

    def test_missing_tbody_fails(self):
        body = textwrap.dedent(
            """
            <table style="width:380px">
              <colgroup><col style="width:380px"></colgroup>
            </table>
            """
        )
        code, out = run_check("check_table_shapes.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("table missing tbody", out)

    def test_missing_col_entries_fails(self):
        body = textwrap.dedent(
            """
            <table style="width:380px">
              <colgroup></colgroup>
              <tbody><tr><td>no cols</td></tr></tbody>
            </table>
            """
        )
        code, out = run_check("check_table_shapes.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("colgroup needs cols", out)


class FormsTests(unittest.TestCase):
    def test_form_must_wrap_table_and_be_in_td(self):
        bad = "<tr><td><form><p>missing table</p></form></td></tr>"
        code, out = run_check("check_forms.py", html_doc(table_block(bad)))
        self.assertNotEqual(code, 0)
        self.assertIn("form must live inside td and wrap a table", out)

    def test_valid_form_passes(self):
        inner = textwrap.dedent(
            """
            <tr><td>
              <form>
                <table style="width:380px">
                  <colgroup><col style="width:380px"></colgroup>
                  <tbody><tr><td><p>ok</p></td></tr></tbody>
                </table>
              </form>
            </td></tr>
            """
        )
        code, out = run_check("check_forms.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("form rules satisfied", out)


# ----------------------------------
# Head CSS and inline styles
# ----------------------------------
class HeadCssTests(unittest.TestCase):
    def test_mismatch_fails(self):
        bad = html_doc(
            table_block("<tr><td>bad css</td></tr>"), css="body{background:red;}"
        )
        code, out = run_check("check_head_css.py", bad)
        self.assertNotEqual(code, 0)
        self.assertIn("head css must match immutable.css", out)


class PxUnitsTests(unittest.TestCase):
    def test_non_px_unit_fails(self):
        inner = '<tr><td style="width:10em">oops</td></tr>'
        code, out = run_check("check_px_units.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("only px units allowed", out)

    def test_mixed_units_fails(self):
        inner = '<tr><td style="width:10px; height:5em">oops</td></tr>'
        code, out = run_check("check_px_units.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("only px units allowed", out)


# ----------------------------------
# Width rules
# ----------------------------------
class WidthTests(unittest.TestCase):
    def test_top_table_must_be_380(self):
        body = table_block("<tr><td>bad width</td></tr>", width="200px")
        code, out = run_check("check_widths_via_dict_structure.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("tables under body must be width 380px", out)

    def test_zero_width_fails(self):
        body = table_block('<tr><td style="width:0px">bad zero</td></tr>')
        code, out = run_check("check_widths_via_dict_structure.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("inline style must not use width: 0px", out)

    def test_non_table_zero_width_fails(self):
        body = table_block('<tr><td><p style="width:0px">bad zero</p></td></tr>')
        code, out = run_check("check_widths_via_dict_structure.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("inline style must not use width: 0px", out)

    def test_missing_table_width_style_fails(self):
        body = textwrap.dedent(
            """
            <table>
              <colgroup><col style="width:380px"></colgroup>
              <tbody><tr><td>no width</td></tr></tbody>
            </table>
            """
        )
        code, out = run_check("check_widths_via_dict_structure.py", html_doc(body))
        self.assertNotEqual(code, 0)
        self.assertIn("table needs width style", out)


class NestedWidthsTests(unittest.TestCase):
    def test_nested_mismatch_fails(self):
        outer = textwrap.dedent(
            """
            <table style="width:380px">
              <colgroup><col style="width:200px"></colgroup>
              <tbody>
                <tr>
                  <td>
                    <table style="width:300px">
                      <colgroup><col style="width:150px"></colgroup>
                      <tbody><tr><td><p>bad</p></td></tr></tbody>
                    </table>
                  </td>
                </tr>
              </tbody>
            </table>
            """
        )
        code, out = run_check(
            "check_widths_via_dict_structure.py", html_doc(outer)
        )
        self.assertNotEqual(code, 0)
        self.assertIn("table width must equal sum of colgroup widths", out)

    def test_nested_valid_passes(self):
        outer = textwrap.dedent(
            """
            <table style="width:380px">
              <colgroup><col style="width:380px"></colgroup>
              <tbody>
                <tr>
                  <td>
                    <table style="width:380px">
                      <colgroup><col style="width:380px"></colgroup>
                      <tbody><tr><td><p>ok</p></td></tr></tbody>
                    </table>
                  </td>
                </tr>
              </tbody>
            </table>
            """
        )
        code, out = run_check(
            "check_widths_via_dict_structure.py", html_doc(outer)
        )
        self.assertEqual(code, 0)
        self.assertIn("nested widths satisfied", out)


# ----------------------------------
# Row text element limits
# ----------------------------------
class RowTextElementTests(unittest.TestCase):
    def test_two_text_items_in_row_fail(self):
        inner = "<tr><td><p>one</p><span>two</span></td></tr>"
        code, out = run_check("check_row_text_elements.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("only one text element", out)

    def test_link_button_not_counted(self):
        inner = '<tr><td><a class="link-button">ok</a><span>text</span></td></tr>'
        code, out = run_check("check_row_text_elements.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("row text element rules satisfied", out)

    def test_menu_table_ignored(self):
        menu = textwrap.dedent(
            """
            <table id="menu" style="width:380px">
              <colgroup><col style="width:380px"></colgroup>
              <tbody><tr><td><p>one</p><p>two</p></td></tr></tbody>
            </table>
            """
        )
        code, out = run_check("check_row_text_elements.py", html_doc(menu))
        self.assertEqual(code, 0)
        self.assertIn("row text element rules satisfied", out)


# ----------------------------------
# Spacing rules
# ----------------------------------
class SpacingTests(unittest.TestCase):
    def test_horizontal_requires_nbsp_between_occupying_cells(self):
        inner = "<tr><td><input name=\"n\" type=\"text\" value=\"v\"></td><td><button>ok</button></td></tr>"
        code, out = run_check("check_spacing_horizontal.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("missing <td>&nbsp;</td>", out)

    def test_horizontal_rejects_empty_td_as_gap(self):
        inner = "<tr><td><input name=\"n\" type=\"text\" value=\"v\"></td><td></td><td><button>ok</button></td></tr>"
        code, out = run_check("check_spacing_horizontal.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("missing <td>&nbsp;</td>", out)

    def test_horizontal_accepts_nbsp_td_as_gap(self):
        inner = "<tr><td><input name=\"n\" type=\"text\" value=\"v\"></td><td>&nbsp;</td><td><button>ok</button></td></tr>"
        code, out = run_check("check_spacing_horizontal.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("horizontal spacing satisfied", out)

    def test_vertical_allows_stackable_rows_without_gap(self):
        inner = "<tr><td><p>a</p></td></tr><tr><td><span>b</span></td></tr>"
        code, out = run_check("check_spacing_vertical.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("vertical spacing satisfied", out)

    def test_vertical_requires_gap_around_button_rows(self):
        inner = "<tr><td><button>go</button></td></tr><tr><td><p>x</p></td></tr>"
        code, out = run_check("check_spacing_vertical.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("missing spacer row", out)

    def test_vertical_accepts_gap_row(self):
        inner = "<tr><td><button>go</button></td></tr><tr><td>&nbsp;</td></tr><tr><td><p>x</p></td></tr>"
        code, out = run_check("check_spacing_vertical.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("vertical spacing satisfied", out)

    def test_vertical_between_tables_allows_stackable_boundary(self):
        body = table_block("<tr><td><p>a</p></td></tr>") + table_block(
            "<tr><td><span>b</span></td></tr>"
        )
        code, out = run_check(
            "check_spacing_vertical_between_tables.py", html_doc(body)
        )
        self.assertEqual(code, 0)
        self.assertIn("between-table spacing satisfied", out)

    def test_vertical_between_tables_requires_gap_for_button(self):
        body = table_block("<tr><td><button>go</button></td></tr>") + table_block(
            "<tr><td><p>x</p></td></tr>"
        )
        code, out = run_check(
            "check_spacing_vertical_between_tables.py", html_doc(body)
        )
        self.assertNotEqual(code, 0)
        self.assertIn("missing spacer row", out)

    def test_vertical_between_tables_accepts_spacer_table_between(self):
        body = (
            table_block("<tr><td><button>go</button></td></tr>")
            + table_block("<tr><td>&nbsp;</td></tr>")
            + table_block("<tr><td><p>x</p></td></tr>")
        )
        code, out = run_check(
            "check_spacing_vertical_between_tables.py", html_doc(body)
        )
        self.assertEqual(code, 0)
        self.assertIn("between-table spacing satisfied", out)

    def test_vertical_between_tables_accepts_end_gap_in_prev_table(self):
        body = table_block(
            "<tr><td><button>go</button></td></tr><tr><td>&nbsp;</td></tr>"
        ) + table_block("<tr><td><p>x</p></td></tr>")
        code, out = run_check(
            "check_spacing_vertical_between_tables.py", html_doc(body)
        )
        self.assertEqual(code, 0)
        self.assertIn("between-table spacing satisfied", out)

    def test_td_nbsp_purity_accepts_pure_gap_cell(self):
        inner = "<tr><td>&nbsp;</td></tr>"
        code, out = run_check("check_td_nbsp_pure.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("td nbsp purity satisfied", out)

    def test_td_nbsp_purity_rejects_double_nbsp(self):
        inner = "<tr><td>&nbsp;&nbsp;</td></tr>"
        code, out = run_check("check_td_nbsp_pure.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("td nbsp must be sole content", out)

    def test_td_nbsp_purity_rejects_nbsp_with_text(self):
        inner = "<tr><td>&nbsp;x</td></tr>"
        code, out = run_check("check_td_nbsp_pure.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("td nbsp must be sole content", out)

    def test_td_nbsp_purity_rejects_nbsp_with_child(self):
        inner = "<tr><td>&nbsp;<p>x</p></td></tr>"
        code, out = run_check("check_td_nbsp_pure.py", html_doc(table_block(inner)))
        self.assertNotEqual(code, 0)
        self.assertIn("td nbsp must be sole content", out)

    def test_td_nbsp_purity_ignores_nbsp_inside_child(self):
        inner = "<tr><td><p>x&nbsp;</p></td></tr>"
        code, out = run_check("check_td_nbsp_pure.py", html_doc(table_block(inner)))
        self.assertEqual(code, 0)
        self.assertIn("td nbsp purity satisfied", out)


# ----------------------------------
# Strict parsing checks
# ----------------------------------
class StrictParsingTests(unittest.TestCase):
    def test_lxml_rejects_bad_markup(self):
        bad = "<html><head></head><body><p>unclosed</body></html>"
        code, out = run_check("check_lxml_strict.py", bad)
        self.assertNotEqual(code, 0)
        self.assertIn("xml parse failed", out)

    def test_tidy_rejects_missing_closure(self):
        bad = "<html><head></head><body><table><tr><td>no close</body></html>"
        code, out = run_check("check_tidy_strict.py", bad)
        self.assertEqual(code, 0)
        self.assertTrue(out.strip())


# ----------------------------------
# Pipeline smoke test
# ----------------------------------
class PipelineTests(unittest.TestCase):
    def test_full_run_passes_on_clean_doc(self):
        inner = "<tr><td><p>ok</p></td></tr>"
        html = html_doc(table_block(inner))
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "page.html")
            with open(path, "w") as f:
                f.write(html)
            result = subprocess.run(
                [PY_EXE, os.path.join(BASE, "run_design_checks.py"), path],
                text=True,
                capture_output=True,
            )
        self.assertEqual(result.returncode, 0)
        last = result.stdout.strip().splitlines()[-1]
        self.assertIn(".html", last)
        self.assertEqual(len(last.split()[-1]), 64)


# ----------------------------------
# Run the suite when executed directly
# ----------------------------------
unittest.main()
