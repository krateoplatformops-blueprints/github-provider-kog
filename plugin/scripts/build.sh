#!/bin/bash

#KO_DOCKER_REPO=kind.local ko build --base-import-paths .
KO_DOCKER_REPO=matteogastaldello ko build -t 1.0.6 --base-import-paths .
printf '\n\nList of current docker images loaded in KinD:\n'

kubectl get nodes kind-control-plane -o json \
    | jq -r '.status.images[] | " - " + .names[-1]'