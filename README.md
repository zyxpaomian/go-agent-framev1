# Rinck_Admin

``` bash
# 开发环境运行
make run 

# 编译安装
make compile
```

# 文件
* conf - 默认配置文件位置
* log - 默认日志文件位置

* src -应用源码
    * command - 可执行文件
    * controller - ctrl层，逻辑编写在这
    * dao - dao层，数据获取
    * service - 基本服务层，底层服务封装，如HTTP,RPC,TCP等
    * structs - 结构体封装
    * util - 基础工具类封装，如DB，日志配置文件等等
* vender - 第三方依赖包
* .Makefile - makefile

# 具体功能
待补充

# License
[MIT](http://opensource.org/licenses/MIT)

# protobuf 包生成
protoc --go_out=../src/msg/   *.proto

# etcd配置路径记录
日志级别: /server/cfg/common/loglevel
日志文件位置: /server/cfg/common/logdir
网络探测周期: /server/cfg/common/detctinterval
Ping网络探测清单: /server/cfg/common/pingiprange
nc网络探测清单: /server/cfg/common/nciprange
tco读超时时间: /server/cfg/common/readtimeout
tcp写超时时间: /server/cfg/common/writetimeout
agent监听地址: /server/cfg/common/agentbind
influxdb监听地址: /server/cfg/influxdb/bind
influxdb用户名: /server/cfg/influxdb/username
influxdb密码: /server/cfg/influxdb/password
抓包吐给服务器: /server/cfg/common/dumpserver
plugin默认位置: /server/cfg/common/plugindir ./plugin/
服务器信息收集清单: /server/cfg/common/collectitems 




