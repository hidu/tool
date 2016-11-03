一些Go写的小工具
====

##xml2json
将xml转换为json，并支持使用json schema来对输出的json进行格式修正  
```
xml2json -xml a.xml -out a.json -jsonschema b.json
```

##xml2json_ser
xml2json web 服务  

##json_indent
json 格式化  
```
cat a.json|json_indent -full
```

## urldecode
```
cat a.log|urldeocde|grep aaa
```

## bdlog_kv
```
cat a.log|bdlog_kv -fs logid,product -h
cat a.log|bdlog_kv -fs http_inpot_post -nokeys|php count.php
```

## url_call_conc
并发的对给出的url list进行请求  
```
cat url_list.txt|url_call_conc -c 100
php ../data/url_call_conc_build.php|url_call_conc -c 100 -complex
```
还可以使用配置文件来动态修改运行的并发量(每30秒load一次)：（./url_call_conc.conf）  
```
{"conc":100}
```
若conc=0 则并发量为0，此时会停止。  
配置文件位于当前运行目录。