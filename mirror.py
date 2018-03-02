import sys
reload(sys)
sys.setdefaultencoding("utf-8")
import json
with open('mirrors', 'r') as f:
    mirrors = json.load(f)

data_cn = []
data_other = []
for data in mirrors['data']:
    if data['location'] == "CN":
        data_cn.append(data)
    else:
        data_other.append(data)

json_cn = {
        "status_code":0,
        "status_message":"ok",
        "data":data_cn
        }

json_other = {
        "status_code":0,
        "status_message":"ok",
        "data":data_other
        }

us_qhcdn = {"id":"qhcdn","weight":100000,"name":"Qhcdn Mirror (CDN Acceleration)","url":"http://us.deepin.qhcdn.com/deepin/","location":"US","locale": {"zh_TW": {"name": "[US] Qhcdn"}, "zh_CN": {"name": "[US] Qhcdn"}}}
json_other['data'].append(us_qhcdn)
with open('mirror_cn.json', 'w') as f:
    json.dump(data_cn, f)

with open('mirror_other.json', 'w') as f:
    json.dump(data_other, f)
