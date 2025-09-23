#!/bin/bash

swag init --parseDependency -g main.go

mkdir -p docs/v3/

scripts/convert_swagger.sh -i docs/swagger.json -o docs/v3/openapi
