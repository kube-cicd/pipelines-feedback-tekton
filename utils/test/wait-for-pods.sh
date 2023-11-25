#!/bin/bash
set -e

podsCreated=false
while [[ $podsCreated == "false" ]]; do
    if [[ $(kubectl get pods "$@" 2>&1) != *"STATUS"* ]]; then
        sleep 1
        continue
    fi
    break
done

exec kubectl wait --for=condition=ready pod "$@"
