#!/bin/bash

GOOS=js GOARCH=wasm go build -o frontend/hackdrive.wasm .