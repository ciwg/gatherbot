#!/bin/bash

set -x

while true
do 
    if ! ./sync.sh 
    then
        sleep 1800
    fi
    sleep 300
done
