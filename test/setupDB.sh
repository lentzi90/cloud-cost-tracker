#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/influx

sudo docker-compose rm -f
sudo docker-compose pull
sudo docker-compose up --build -d
