#!/usr/bin/python

import sys
import json

data = json.load(sys.stdin)
node_name = sys.argv[1]
print('<pre id="json">' + json.dumps({ node_name: data["nodes"][node_name]}, indent=2) +'</pre>')