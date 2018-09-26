#!/bin/bash
set -x

rm result_*|| echo "result file not exist"

#1. 获取镜像源列表。从mirror list CMS里获取数据
$WORKSPACE/lastore-tools update -j unpublished-mirrors --mirrors-url http://server-12:8900/v1/mirrors -o mirrors
#1.1 拆分mirrors为国内和国外两个列表.
python mirror.py

#2. 使用lastore-tools检测同步进度。TODO，需要剥离lastore-tools的此功能到本项目
NAME_SUFFIX=$(date +%F_%T)
stdbuf -o0 $WORKSPACE/lastore-tools smartmirror  -m  "$WORKSPACE/mirror_cn.json" server_stats -e result_cn_$NAME_SUFFIX.json $server

#TODO，实际这里应该通过国外节点进行检测，但相关代码已经被注释
#http_proxy=10.0.0.42:8888 stdbuf -o0 $WORKSPACE/lastore-tools smartmirror  -m  "$WORKSPACE/mirror_other.json" server_stats $server
stdbuf -o0 $WORKSPACE/lastore-tools smartmirror  -m  "$WORKSPACE/mirror_other.json" server_stats -e result_other_$NAME_SUFFIX.json $server

rm mirrors
rm mirror_*

#3. 分析result_xxx.json并汇报到trend.deepin.io
#CI上需要配置INFLUX_USER，INFLUX_PASSWD这两个关键信息。
#关闭输出，避免敏感信息被echo
set +x
./push_to_influxdb -db mirror_status -host http://influxdb.trend.deepin.io:10086 -user $INFLUX_USER -password $INFLUX_PASSWD result_cn_$NAME_SUFFIX.json result_other_$NAME_SUFFIX.json
set -x
