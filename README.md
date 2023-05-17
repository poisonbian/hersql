# 使用说明

## 原始代码：
https://blog.fanscore.cn/a/47/

https://github.com/polarrwl/hersql

但是代码有比较多的问题，虽然可以编译执行，但是用客户端连接、使用的时候，有不少错误。因此对整个代码又进行了若干修复后，进行发布。

## 解决的问题
内网、虚拟主机等环境下，mysql通常无法直接对外访问，也就是说，Navicat、DBeaver等客户端可能用不了了。虽然可以使用phpmyadmin等方式来进行网页访问，但使用起来还是比较麻烦。

Navicat提供了一个基于php的mysql_ntunnel.php小工具，可以提供http代理的功能。也就是说，将这个php代码部署在对应环境后，在Navicat中新建连接时，设置使用http方式连接，就可以在客户端中进行数据库管理。

但是，这个工具不支持其他mysql客户端，比如DBeaver。

## 思路
1. 提供一个本地可执行工具，运行后，可以mock成一个mysql server，开启运行
2. 这个mock server接受到db访问请求后，通过http连接的方式，访问内网/虚拟主机暴露出来的mysql_ntunnel.php

## 方案
可能这个场景的需求比较小众吧，好不容易在网上找到了hersql这个开源代码。但是clone下来之后，发现无法运行。

go代码不太熟悉，边学边调试，一步步去解决遇到的问题

### 问题1
刚开始运行的时候，DBeaver、Navicat都提示了类似的错误：
no dsn specified before the query:..., you can try restarting the mysql client

看起来是在执行sql语句前，没有初始化“DSN”信息。从代码看，这个DSN信息在执行了“USE”语句的时候（比如选择了哪个数据库）会自动执行。可是DBeaver、Navicat，到底在连接数据库的时候，执行哪些语句，这个我们无法控制。

所以我的方案是，不管怎么样，先默认执行一个DSN的初始化。

因此又改了下配置，增加了数据库的初始信息。（这一点也很奇怪，没有配置这个信息的话，原来到底是怎么执行起来的呢？）

OK，改完之后，可以了。

### 问题2
在DBeaver中操作时，有时候会弹出提示，类似于：

SQL 错误 [1105] [HY000]: unknown error: Reader.parseBlockValueWithFirstByte read value n:[1] != [19]

数据内容也无法展示。

这个报错是hersql里面报出来的，通过http请求获得response之后进行解析的过程中会报错。

实际上这个报错还挺多的，主要是在返回数据过多过长的时候就会报。

于是又系统地学习了一下go语言里面http请求、reader之类的知识，把response打印出来一点点定位。

经过艰苦地定位，并没有找到原因，但是。。。莫名其妙地好了-____-!!!

## 使用方式
### 部署ntunnel_mysql.php
即项目中的_mysql.php 文件，部署至内网，或其他可以访问对应数据库的机器上。并保证其能够通过一个http地址进行访问

验证：直接在浏览器中访问http地址，在页面中输入对应的数据库信息后，测试是否可以联通。

### 准备hersql
1. 安装go环境、下载代码

2. 编译（也可以使用compile.sh，自动编译win/mac/linux多平台的执行文件）

3. 配置conf文件，可以参照conf.yml.example

4. 运行：./hersql.exe -conf xxxconf.yml

5. 在navicat或者dbeaver等客户端软件中配置连接即可

### 配置说明
\# 数据库对应信息。这些信息会传给http代理，供其连接真正的数据库
db_info:
  ntunnel_url: http://test.navicat.com/ntunnel_mysql.php
  host: xxdb.xx.com
  port: 3306
  database: a_default_db
  user: username
  password: password
\# hersql启动的server信息。这里的user_name和user_password用于navicat中的配置
server: 
  protocol: tcp
  address: 127.0.0.1:63306
  version: 5.7.1
  conn_read_timeout: 300000 # milliseconds
  conn_write_timeout: 5000 # milliseconds
  max_connections: 10
  user_name: root
  user_password: 123456
\# 日志记录
log:
  info_log_filename: storage/log/info.log
  error_log_filename: storage/log/error.log


