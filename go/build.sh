#!/bin/bash
cd $(dirname $0)

go build -o ../dest/bin/json_indent -ldflags "-s -w"  json_indent.go 
#go build -o ../dest/bin/qqwry -ldflags "-s -w"  qqwry.go 
go build -o ../dest/bin/mysqldbtest -ldflags "-s -w"  mysqldbtest.go 
go build -o ../dest/bin/xml2json -ldflags "-s -w"  xml2json.go 
go build -o ../dest/bin/xml2json_ser -ldflags "-s -w"  xml2json_ser.go 
go build -o ../dest/bin/httptest -ldflags "-s -w"  httptest.go 
go build -o ../dest/bin/urldecode -ldflags "-s -w"  urldecode.go 
go build -o ../dest/bin/bdlog_kv -ldflags "-s -w"  bdlog_kv.go 
go build -o ../dest/bin/httpserver4test -ldflags "-s -w"  httpserver4test.go 
