#!/usr/bin/env python
#coding=utf-8

import socket, struct
import sys
for line in sys.stdin:
    print socket.inet_ntoa(struct.pack('!L', long(line.strip())))