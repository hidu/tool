tool
====


##xml2json
convert xml to json and fix data with json schema  
```
xml2json -xml a.xml -out a.json -jsonschema b.json
```

##xml2json_ser
simple xml2json server

##json_indent
```
cat a.json|json_indent -full
```

##urldecode
```
cat a.log|urldeocde|grep aaa
```

##bdlog_kv
```
cat a.log|bdlog_kv -fs logid,product -h
cat a.log|bdlog_kv -fs http_inpot_post -nokeys|php count.php
```
