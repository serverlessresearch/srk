import json
import time
import urllib.request
import urllib
import resource

from uuid import uuid4

lambda_id = str(uuid4())

class LambdaExperiment(object):
    def __init__(self, uuid, base_url):
        self.uuid = uuid
        self.base_url = base_url.rstrip('/') + '/'
        self.begin_time = None
        self.end_time = None

    def _url(self, path):
        return urllib.parse.urljoin(self.base_url, path)

    def begin(self):
        data = str.encode(json.dumps({
            'lambda_id': lambda_id,
            'action': 'begin',
            'uuid': self.uuid
        }))
        req = urllib.request.Request(self._url('event'), data)
        response = urllib.request.urlopen(req)
        print(response.read())
        self.begin_time = time.time()
        # todo - decide whether its ok to proceed

    def end(self):
        self.end_time = time.time()
        data = str.encode(json.dumps({
            'lambda_id': lambda_id,
            'action': 'end',
            'uuid': self.uuid,
            'begin_time': self.begin_time,
            'end_time': self.end_time
        }))
        req = urllib.request.Request(self._url('event'), data)
        response = urllib.request.urlopen(req, data)
        print(response.read())

    def post_data(self, experiment_data):
        data = str.encode(json.dumps({
            'lambda_id': lambda_id,
            'action': 'report',
            'uuid': self.uuid,
            'data': experiment_data
        }))
        req = urllib.request.Request(self._url('data'), data)
        response = urllib.request.urlopen(req)
        print(response.read())


def lambda_handler(event, context):
    exp = LambdaExperiment(event['uuid'], event['tracking_url'])

    exp.begin()
    begin_rusage = resource.getrusage(resource.RUSAGE_SELF)
    sleep_time_ms = event['sleep_time_ms']
    if not isinstance(sleep_time_ms, int):
        return {
            'statusCode': 400,
            'body': json.dumps('Invalid sleep time')
        }
    time.sleep(sleep_time_ms/1000)
    exp.end()
    end_rusage = resource.getrusage(resource.RUSAGE_SELF)
    exp.post_data({
        'begin_rusage': begin_rusage,
        'end_rusage': end_rusage
    })

    return {
        'statusCode': 200,
        'body': json.dumps('success')
    }