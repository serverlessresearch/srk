#!/bin/bash
# This script installs all of the runtime components of SRK to a central location (default ~/.srk).

echo "Please specify an install location (or press enter to default to ~/.srk)"
read -e SRKHOME

if [[ $SRKHOME == "" ]]; then
    SRKHOME=~/.srk
fi

# OSX does not support readlink -f so we have to resort to this roundabout method
SRKHOME=$(python3 -c "import os; print(os.path.realpath(os.path.expanduser('$SRKHOME')))")

mkdir -p $SRKHOME
cp -r ./runtime $SRKHOME

echo "SRK installed to $SRKHOME"
echo "Please add $SRKHOME/config.yaml and configure for your needs"
echo "You should add \"export SRKHOME=$SRKHOME\" to your .bashrc or equivalent"
