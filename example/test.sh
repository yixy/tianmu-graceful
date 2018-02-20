#!/bin/bash

#./test.sh 2>&1 >test

for k in $(seq 1 500)
do
	curl http://localhost:9999/example?a=$k &
done
