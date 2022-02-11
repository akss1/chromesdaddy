#!/usr/bin/env sh

cron

/usr/local/bin/chromebalancer &
child=$!

trap "kill $child" TERM INT
wait "$child"
trap - TERM INT
wait "$child"
