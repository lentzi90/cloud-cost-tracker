#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
echo "Setting GOPATH to \"$DIR\""
export GOPATH=$DIR

echo "Downloading Azure SDK..."
go get -u github.com/Azure/azure-sdk-for-go/...
echo "Downloading InfluxDB Client..."
go get -u github.com/influxdata/influxdb/client/v2
