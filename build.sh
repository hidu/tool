#!/bin/bash
set -x
cd $(dirname $0)
mkdir -p dest/bin/

go build -o dest/bin/json_indent -ldflags "-s -w"  json_indent/json_indent.go 
#go build -o dest/bin/qqwry -ldflags "-s -w"  qqwry.go 
go build -o dest/bin/mysqldbtest -ldflags "-s -w"  mysqldbtest/mysqldbtest.go 
go build -o dest/bin/xml2json -ldflags "-s -w"  xml2json/xml2json.go 
go build -o dest/bin/xml2json_ser -ldflags "-s -w"  xml2json_ser/xml2json_ser.go 
go build -o dest/bin/httptest -ldflags "-s -w"  httptest/httptest.go 
go build -o dest/bin/urldecode -ldflags "-s -w"  urldecode/urldecode.go 
go build -o dest/bin/bdlog_kv -ldflags "-s -w"  bdlog_kv/bdlog_kv.go 
go build -o dest/bin/url_call_conc -ldflags "-s -w"  url_call_conc/url_call_conc.go
go build -o dest/bin/bdlog_filter -ldflags "-s -w"  bdlog_filter/bdlog_filter.go 
