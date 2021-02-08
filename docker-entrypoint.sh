#!/bin/sh

set -x

echo "starting zerotier"
setsid /usr/sbin/zerotier-one &

while ! pgrep -f zerotier-one
do
  echo "zerotier hasn't started, waiting a second"
  sleep 1
done

echo "joining networks"

for i in "$@"
do
  echo "joining $i"

  while ! zerotier-cli join "$i"
  do 
    echo "joining $i failed; trying again in 1s"
    sleep 1
  done
done

sleep infinity
