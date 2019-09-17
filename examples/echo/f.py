# This is a compatiblity shim for Open Lambda. Eventually, we will likely make
# OpenLambda compatible with AWS Lambda and remove the need for this
import echo
def f(event):
    return echo.echo(event)
