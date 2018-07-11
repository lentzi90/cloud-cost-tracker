#!/bin/bash

if [ "$EUID" -ne 0 ]
  then echo "Must be ran as root"
  exit
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/test/influx

sudo docker-compose kill