#!/bin/bash
cd $(dirname $0)
cd ../
#protoc --plugin=grpc --go_out=. *.proto
protoc  -I ./kafka ./kafka/kafka.proto --go_out=plugins=grpc:kafka
