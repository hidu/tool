# comm 文件对比工具

## 1.基本功能

文件a的一列(a.m) 和文件b的一列(b.n)做对比（前提：两文件已经按照上述a.m 以及 b.n 排序为有序）,比对出 `a.m 包含在b.n中` 或者 `a.m不包含在b.n 中`。

## 2.示例
<table>
    <tr>
        <td>文件a</td>
         <td>文件b</td>
    </tr>
    <tr>
        <td><pre>1  a
1   b
10  f
100 r
11  e
11  e
11  q
12  e
13  f
2   c
2   d
5   e
8   f
8   g
8   h
9   h
99  s
99  t</pre>
</td>
        <td><pre>1  2
1   w
100 3
2   2
3   2
5   2
5   2
5   2
5   3
6   22
6   3
6   3
8   3
99  3</pre></td>
    </tr>
</table>

```
$ fcomm -fac 0 -fbc 0 a.sort b.sort
```
输出


```
1   a
1   b
100 r
2   c
2   d
5   e
8   f
8   g
8   h
99  s
99  t
```