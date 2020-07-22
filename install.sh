#!/bin/bash
# This script installs all of the runtime components of SRK to a central location (default ~/.srk).

# OSX does not support readlink -f so we have to resort to this roundabout method
function abspath {
    python3 -c "import os; print(os.path.realpath(os.path.expanduser('$1')))"
}

RUNTIMEDIR=$(abspath ./runtime)

if [[ -z $SRKHOME ]]; then
    echo "Please specify an install location (or press enter to default to ~/.srk)"
    read -e SRKHOME

    if [[ $SRKHOME == "" ]]; then
        SRKHOME=~/.srk
    fi

    SRKHOME=$(abspath $SRKHOME)
fi

if [[ $SRKHOME != $RUNTIMEDIR ]]; then
    mkdir -p $SRKHOME
    cp -r $RUNTIMEDIR/* $SRKHOME
fi

echo "SRK installed to $SRKHOME"
echo "Please add $SRKHOME/config.yaml and configure for your needs"
echo "You should add \"export SRKHOME=$SRKHOME\" to your .bashrc or equivalent"
