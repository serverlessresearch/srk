#!/bin/bash
# This script installs all of the runtime components of SRK to a central location (default ~/.srk).

echo "Please specify an install location (or press enter to default to ~/.srk)"
read -e SRKHOME

if [[ $SRKHOME == "" ]]; then
    SRKHOME=~/.srk
fi
SRKHOME=$(readlink -f $SRKHOME)

cp -r ./runtime $SRKHOME

echo "SRK installed to $SRKHOME"
echo "Please add $SRKHOME/config.yaml and configure for your needs"
echo "You should add \"export SRKHOME=$SRKHOME\" to your .bashrc or equivalent"
