#!/bin/bash

###############################################################################
#  warp the goimports tool.
#  format the imports as 3 parts.
#  the current project'package always at the last.
#
#   author: github.com/hidu
#   since: 2019年11月21日
#   useage:
#      goimports.sh code_path.go
###############################################################################

if [ ! -f "$1" ];then
    echo "file required"
    exit 1
fi

function formatImport() {
    FullName=`readlink -f  $1`
    
    echo "$FullName"
    
    if [ "${FullName##*.}"x != "go"x ];then
       echo "not go file"
       return
    fi
    
    // 获取当前文件所在package,域名后面取3层目录
    PKG=`echo "$FullName"|grep -oP '(?<=src\/)(\w+(\.\w+)+\/\w+\/\w+\/\w+)'`
    
    // PKG 取值，如  github.com/hidu/abc/def/
    echo "package: $PKG"
    
    if [ -z "$PKG" ];then
        echo "PKG is emptu"
        return
    fi
    
    set -x
    
    goimports -w -local "$PKG" "$FullName"
}

formatImport "$1"
