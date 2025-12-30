Design checks (rendered HTML):

Command:
TMPDIR=/dev/shm uv run python3 run_design_checks.py /dev/shm/almono_requests.html
Output:
/dev/shm/almono_requests.html 0c38d4343c54276865d453195cd11fd2be407708b289cde4d41fd7be432b9ca6

Command:
TMPDIR=/dev/shm uv run python3 run_design_checks.py /dev/shm/almono_request_create.html
Output:
/dev/shm/almono_request_create.html 2dad7b3170dcd1c09eb6758c24d6e54129b95b6bb83b2876ff94bfd1b170914c

Command:
TMPDIR=/dev/shm uv run python3 run_design_checks.py /dev/shm/livestream.html
Output:
/dev/shm/livestream.html 0649a3c9d59cbad599b500393345c12eece8f29ab5eb25878487a6a553a2329e

Command:
python3 run_design_checks.py /dev/shm/almono_request_cast.html
Output:
/dev/shm/almono_request_cast.html 0063953231dc8453d7b78cf3013bb3aed95aab187a9bce9f2b6d5a6fbbf947f1
