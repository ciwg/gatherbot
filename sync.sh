#!/bin/sh

set -ex

# change each week
export GATHER_SPACE_ID="$(cat local/space-id-day1)"

dir=/tmp/gatherbot.json
mkdir -p $dir

dayfn=$1
if [ -z "$dayfn" ]
then
    ls -l $dir
    exit 1
fi

export EVENTBRITE_EVENT="$(cat local/eventbrite.event-id)"
export EVENTBRITE_AUTH="$(cat local/eventbrite.key)"
export GATHER_API_KEY="$(cat local/gather-api-key)"

go run main.go api csv
go run main.go api json

# avlaptop=192.168.86.30
avlaptop=daphne.local
rsync -avz /tmp/gatherbot.* $avlaptop:/tmp

echo "following will fail if guest list is empty; if so, run with GATHER_OVERWRITE=true to seed it" 

curl -i -H "Content-Type: application/json" --data @$dir/$dayfn 'https://api.gather.town/api/setEmailGuestlist'
echo
date
