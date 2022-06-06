#!/bin/bash

day=$1

while true
do 
    ./sync.sh $day
    sleep 300
done
