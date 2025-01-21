#!/bin/bash

curl https://cloud.tuist.io/api/spec | jq 'del(.security, .components.securitySchemes)' > openapi.json
