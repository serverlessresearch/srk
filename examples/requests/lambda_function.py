import requests

def lambda_handler(event, context):
    response = requests.get('https://httpstat.us/200')
    if response.status_code == 200:
        return { 'success': True }
    else:
        return { 'success': False }
