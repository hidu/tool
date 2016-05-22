#!/usr/bin/env python
#coding=utf-8

import argparse
import sys

def parse_args():
    description ="shuffle and merge files"
    parser = argparse.ArgumentParser(description = description)
    parser.add_argument('input_files',nargs = '*')
    parser.add_argument('-n',type=int, default=0,help="output file total,0:same sa input_files total")
    parser.add_argument("-o","--prefix",type=str,help="output file prefix")
    
    args = parser.parse_args();
    return args
    
def main():
    args = parse_args()
#     print("args:",args)
    
    if(len(args.input_files)==0):
        print "empty input_files"
        sys.exit(2)
    if (args.prefix==None or args.prefix==""):
        print "prefix required"
        sys.exit(2)
        
    input_fs={}
    
    for name in args.input_files:
        try:
            fh=open(name,"r")
#             print("open name:",name)
            input_fs[name]=fh
        except IOError, e:
            print e
            sys.exit(2)
        
        
    outTotal=args.n if args.n>0  else len(args.input_files)
    
    out_fs=[]
    for i in range(0,outTotal):
        try:
            fh=open(args.prefix+str(i),"w+")
            out_fs.append(fh)
        except IOError, e:
            print e
            sys.exit(2)
           
    merge(input_fs, out_fs)
    
    for i in range(0,outTotal):
        fh=out_fs[i]
        fh.close()
        
       
def merge(input_fs,out_fs):
    num=0
    out_nums=range(0,len(out_fs))
    while len(input_fs)>0:
        for idx in out_nums:
            for name in input_fs.keys():
                fh=input_fs[name]
                line=fh.readline()
                if line:
                    fh=out_fs[idx]
                    fh.write(line)
                else:
                    input_fs.pop(name)
    
if __name__ == '__main__':
    main()