import cfbench
import json
import logging
import resource
import time
import urllib.request
import urllib

logger = logging.getLogger()
logger.setLevel(logging.INFO)

def lambda_handler(event, context):
    with cfbench.LambdaExperiment(event, context) as exp:
        begin_rusage = resource.getrusage(resource.RUSAGE_SELF)

        sleep_time_ms = event['sleep_time_ms']
        if not isinstance(sleep_time_ms, int):
            return {
                'statusCode': 400,
                'body': json.dumps('Invalid sleep time')
            }

        time.sleep(sleep_time_ms/1000)

        end_rusage = resource.getrusage(resource.RUSAGE_SELF)

        exp.data({
            'begin_rusage': begin_rusage,
            'end_rusage': end_rusage
        })

        return {
            'statusCode': 200,
            'body': json.dumps('success')
        }