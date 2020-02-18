#!/bin/bash
set -x
set -e

# Hack to let us call this like ./tools/test.sh or "cd test; ./test.sh"
if [ -f ../srk ]; then
  cd ..
fi

echo "Rebuilding"
go build

echo "Create Func"
./srk function create --source examples/echo

echo "Run Bench"
./srk bench \
        --benchmark one-shot \
        --function-args '{"hello" : "world"}' \
        --function-name echo

echo "Remove Func"
./srk function remove -n echo
