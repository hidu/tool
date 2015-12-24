#!/bin/env python
#coding=utf-8

#urldecode strings
#eg
# $echo '%E4%BD%A0%E5%A5%BD'|./urldecode.py 
# $你好

import sys
from urllib import unquote
for line in sys.stdin:
    print unquote(line);
