qps 实时统计
====

##USAGE

a.log
```
read   10
read   10
read   10
write   20
```


```
$ cat a.log|qps_count
$ tail -f a.log|qps_count
```

```
cat a.log|qps_count -k 1 -v 2
```