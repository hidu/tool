#!/bin/env perl

#urldecode strings
#eg
# $echo '%E4%BD%A0%E5%A5%BD'|./urldecode.pl 
# $你好

use URI::Escape;
while(<>){
   print uri_unescape($_);
}