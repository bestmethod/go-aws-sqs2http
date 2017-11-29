'''
Quick server example, written in python with flask.
Used to test full flow in live environment.
Will allow for test to pass and will print json received with message.
'''

from flask import Flask
from flask import request
import json
app = Flask(__name__)

@app.route('/sqs/test', methods=['GET'])
def ok_200():
    return 'Hello, World!',200

@app.route('/sqs/newMessage', methods=['POST'])
def print_post_content():
    print json.dumps(request.json)
    return 'OK',200

app.run()
