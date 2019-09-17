import json
import logging
import resource
import time
import urllib.request
import urllib

logger = logging.getLogger()

from uuid import uuid4

lambda_id = str(uuid4())

class LambdaExperiment(object):
    def __init__(self, event, context):
        if 'uuid' in event:
            self.uuid = event['uuid']
        else:
            logger.info('no uuid provided so generating one')
            self.uuid = str(uuid4())
        logger.info('initializing experiment with uuid %s' % self.uuid)
        if 'tracking_url' in event:
            self.base_url = event['tracking_url'].rstrip('/') + '/'
            logger.info("using tracking url %s" % self.base_url)
        else:
            self.base_url = None
            logger.info("no tracking url provided.")
        self.begin_time = None
        self.end_time = None
        self.data_buffer = {}

    def _url(self, path):
        return urllib.parse.urljoin(self.base_url, path)

    def __enter__(self):
        data = json.dumps({
            'lambda_id': lambda_id,
            'action': 'begin',
            'uuid': self.uuid
        })
        logger.info(data)
        if self.base_url:
            req = urllib.request.Request(self._url('event'), str.encode(data))
            response = urllib.request.urlopen(req)
            # TODO start workload after specific time delay
            self.begin_time = time.time()

        return self

    def __exit__(self, type, value, tb):
        # TODO report exceptions
        self.end_time = time.time()
        data = json.dumps({
            'lambda_id': lambda_id,
            'action': 'end',
            'uuid': self.uuid,
            'begin_time': self.begin_time,
            'end_time': self.end_time
        })
        logger.info(data)
        if self.base_url:
            req = urllib.request.Request(self._url('event'), str.encode(data))
            response = urllib.request.urlopen(req)

        data = json.dumps({
            'lambda_id': lambda_id,
            'action': 'report',
            'uuid': self.uuid,
            'data': self.data_buffer
        })
        logger.info(data)
        if self.base_url:
            req = urllib.request.Request(self._url('data'), str.encode(data))
            response = urllib.request.urlopen(req)


    def data(self, experiment_data):
        self.data_buffer.update(experiment_data)

