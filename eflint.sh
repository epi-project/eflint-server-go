#!/bin/bash

# Run eflint-to-json
json=$(./eflint-to-json "$1")

# Check for errors
if [ $? -ne 0 ]; then
    echo "Error running eflint-to-json on $1"
    exit 1
fi

curl --location --request GET 'http://localhost:8080' --header 'Content-Type: application/json' --data-raw "$json" | python3 -m json.tool