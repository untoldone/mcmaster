#!/bin/bash

set -e

cd frontend

yarn run dev &

cd ../backend

go run .
