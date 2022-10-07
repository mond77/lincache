#!/bin/bash
trap "rm server;kill 0" EXIT

go build -o server
./server -port=8001 &
./server -port=8003 &
./server -port=8002 -api=1 &

sleep 2
echo ">>> start test"

curl "http://localhost:9999/api?key=Tom" &
sleep 1
curl "http://localhost:9999/api?key=Jack" &

curl "http://localhost:9999/api?key=Sam" &
curl "http://localhost:9999/api?key=Sam" &
curl "http://localhost:9999/api?key=Tom" &

sleep 1
curl "http://localhost:9999/api?key=Sam" &

sleep 1
curl "http://localhost:9999/api?key=Tom" &

sleep 1
curl "http://localhost:9999/api?key=Jack" &

wait