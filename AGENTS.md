### Aggresively reduce count_bytecode.py score as much as possible for python code.

- In each project, run `python3 count_bytecode.py <py_script>` on the py script that you create or change and it gives a score. We have to reduce this score aggresively.
- Do not put code in text and use 'exec' on it to reduce score, thats cheating.
- Record the score in score.md file.

### Python Module docstring

- Check that we must have 'MUST HAVE REQUIREMENTS' section at top of file.
- Code logic should never miss any 'MUST HAVE REQUIREMENTS'

## Tips to reduce bytecode score for python.

1. Remove unecssary instructions, like validations or checks, let it fail, thats ok because our priority is reducing bytecode score.
2. Using dictionary direct indexing rather than .get() attribute. For example dont use os.environ.get() or os.getenv() instead use os.environ["key"]
3. When in doubt whether to remove code or not always remove it, user can always ask you to put the code back if really needs it. Your goal is to aggresively lower the count_bytecode score.
4. If the script is standalone and not imported elsewhere, there is no need to have main() function.
5. Feel free to execute any piece of code with count_bytecode.py to check for its score. This helps to know whether score gets reduced or increased. Use below format if you just want to check score for adhoc pieces of code.
```py
python3 count_bytecode.py <<'PYCODE'
print("hi")
PYCODE
```
6. Do not compromise on naming convention because it doesn't contribute to bytecode score, having good naming is essential for code readability.

7. Favor literal definitions when possible—building lists/tuples/dicts in place avoids extra assignments.
8. Inline expressions and unpacking beats multiple temporary variables in tight loops; each additional instruction can raise the count.
10. Reject unused imports or helper functions; every definition creates more bytecode even if never executed.
s Comments should stay succinct; AGENTS.md already expects block separators, so only add them when they explain behavior necessary for understanding the lean code.
11. Omit wrapping logic in a `main()` if the file is a one-shot script—each standalone function adds another code object the bytecode counter sees, so simpler equals fewer instructions.
12. Dont use __name__ == '__main__', because these scripts will never be imported by others.
13. Dont access environment variables directly, either pass them as cmd line args or use django settings if inside django app.

### Code readability and comments for python code.

1. Add more comments, especially block level comments that visually show what a particular block does.
2. Comments and blank lines do not add bytecode; feel free to use them so long as they stay focused and concise.
3. Use meaningful naming convention in code.
4. Add new lines where necessary to improve code readability.

## Design checks for web
- For alui, we have to run `python3 run_design_checks.py` on rendered html files, our html files will be jinja2 templates so use some random data or curl the endpoint to render them first and then do design checks. Include the run output in design_checks.md.

