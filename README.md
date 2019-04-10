## Start
+ Adjust *config.yml* in configs 
  ```yaml
  influxdb:
       host: 
       port: 
       dbName: 
       username: 
       password: 
  http:
       port: 
  ``` 

+ Add dependencies
  ```shell
	 go mod vendor
  ```
## Run
- make run
## Notice
在 https://ci.deepin.io/job/mirror_status 中，配置源码为本项目，并设置执行脚本为ci.sh

具体行为，参考ci.sh
