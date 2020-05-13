#!/bin/bash

function cleanup {
  kill $GO_PID
  kill $YARN_PID
}

set -e

if [[ -f "$(pwd)/.env" ]]; then
  echo "SOURCING"
  source $(pwd)/.env
fi

trap cleanup SIGHUP SIGINT SIGTERM

cd $(pwd)/frontend

yarn run dev &
YARN_PID=$!

cd ../backend

go run . &
GO_PID=$!

wait $GO_PID
wait $YARN_PID
