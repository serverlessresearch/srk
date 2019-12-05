# This is a bit of a hack for openlambda
import subprocess as sp
import sys

def f(event):
    p = sp.run('ls /dev/', stdout=sp.PIPE, universal_newlines=True, shell=True)
    if p.returncode != 0:
        return "Failed execution"
    else:
        return p.stdout
