## Start
+ Adjust *config.yml* in configs 
  ```yaml
  http:
    port: 
  
  influxdb:
    host: 
    port: 
    dbName: 
    password:
    username:
  
  mysql:
    host: 
    port: 
    dbName: 
    username: 
    password: 
  
  cdn-checker:
    default-cdn: 
    user-agent: 
    api-site: 
    api-path: 
    target: 
    source-url: 
    source-path: 
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
