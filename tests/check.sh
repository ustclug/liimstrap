#!/bin/bash
set -e
cd "$(dirname "$0")/.."
shellcheck liimstrap deploy docker-run.sh bin/*.sh tests/*.sh
for script in liimstrap deploy docker-run.sh bin/*.sh tests/*.sh; do
  bash -O extglob -n "$script"
done
python3 -m unittest discover -s tests -p 'test_*.py'
git diff --check
