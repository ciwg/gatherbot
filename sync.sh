#!/bin/sh

set -ex

mkdir -p /tmp/gatherbot.json /tmp/gatherbot.csv

now=$(date '+%s')
cp -a /tmp/gatherbot.csv /tmp/gatherbot.csv.$now
cp -a /tmp/gatherbot.json /tmp/gatherbot.json.$now

export EVENTBRITE_EVENT="$(cat local/eventbrite.event-id)"
export EVENTBRITE_AUTH="$(cat local/eventbrite.key)"
export GATHER_API_KEY="$(cat local/gather-api-key)"

export GATHERBOT_CONF=local/gatherbot-conf.json

go run main.go api csv
go run main.go api json

# avlaptop=192.168.86.30
# avlaptop=daphne.local
# rsync -avz /tmp/gatherbot.* $avlaptop:/tmp

echo "following will fail if guest list is empty; if so, run with Overwrite=true in conf file to seed it" 

# XXX generate curl commands from Go, one curl for each configured day
# XXX or just send json from Go -- https://www.google.com/search?q=golang+send+json+http+client&num=50&safe=off&tbs=li:1
# XXX e.g. https://stackoverflow.com/a/24455606/1264797

bash -ex /tmp/gatherbot.json/cmds.sh > /tmp/gatherbot.json/out 2>&1
grep HTTP /tmp/gatherbot.json/out

echo
date
