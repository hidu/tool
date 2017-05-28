# fcomm 文件对比工具

## 1.概述
### 1.1 基本功能

文件a的一列(a.m) 和文件b的一列(b.n)做对比（前提：两文件已经按照上述a.m 以及 b.n 排序为有序）,比对出 `a.m 包含在b.n中` 或者 `a.m不包含在b.n 中`。  

相当于:
```sql
select a.* from a where a.m in (select b.n from b)
```

OR
```sql
select a.* from a where a.m not in (select b.n from b)
```

### 1.2 安装
```bash
go get github.com/hidu/tool/fcomm
```


## 2.示例

### 2.1 筛选出文件a  第一列 存在于 文件b第一列 的所有行
```bash
$ fcomm -fac 1 -fbc 1 a.sort b.sort
```
相当于
```sql
select a.* from a where a.1 in (select b.1 from b)
```


<table>
<tr><td>a.sort</td><td>b.sort</td><td>result</td></tr>
<tr>
<td valign=top>
<pre>1  a
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
<td valign=top>
<pre>1  2
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
99  3</pre>
</td>
<td valign=top>
<pre>1  a
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
</pre>
</td>
</tr>
</table>

### 2.2 筛选出文件a第一列 不在 文件b第一列 的所有行
```bash
$ fcomm -fac 1 -fbc 1 -a_in_b=false a.sort b.sort
```
相当于
```sql
select a.* from a where a.1 not in (select b.1 from b)
```

<table>
<tr><td>a.sort</td><td>b.sort</td><td>result</td></tr>
<tr>
<td valign=top>
<pre>1 a
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
<td valign=top>
<pre>1  2
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
99  3</pre>
</td>
<td valign=top>
<pre>10 f
11  e
11  e
11  q
12  e
13  f
9   h
</pre>
</td>
    </tr>
</table>

### 2.3 筛选出文件a 第一列 存在于 文件b第一列 的所有行 以及b的所有行
```bash
$ fcomm -fac 1 -fbc 1 -concat_b a.sort b.sort
```
大致相当于
```sql
select a.match_line,b.first_match_line from a where a.1 in (select b.1 from b)
```
匹配行数 不是 a * b，而是等于所有a匹配的行数。


<table>
<tr><td>a.sort</td><td>b.sort</td><td>result</td></tr>
<tr>
<td valign=top>
<pre>1  a
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
 <td valign=top>
<pre>1  2
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
99  3</pre>
</td>
<td valign=top>
<pre>1  a    1  2
1   b    1  2
100 r    100    3
2   c    2  2
2   d    2  2
5   e    5  2
8   f    8  3
8   g    8  3
8   h    8  3
99  s    99 3
99  t    99 3
</pre>
</td>
    </tr>
</table>
