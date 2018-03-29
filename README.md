### Sync-Dev
实时 SFTP 同步工具（单向）：

* 监听控本地目录文件变更，并通过 SFTP 同步到远端文件服务；
* 完整覆盖同步支持；
* 支持监听一次性监听本地多个路径并同步到远程对应路径；

可以用于在 本地编写代码，并将代码同步到远程 Linux 服务器调试运行等。

### 使用说明
本工具支持两种使用配置使用方式：

1. 命令行单个监听同步：
	``` BASH
	sync-dev -l D:\data\htdocs -r wuhao:123456@11.22.33.44:22/data/wuhao
	```
	附加 `-s` 参数则立即进行完整同步，同步完成后退出（参考参数说明）；
	若不提供密码，则自动使用当前用户私钥进行连接登录（可用 HOME 环境变量覆盖用户路径，即：${HOME}/.ssh/id_rsa）
	
2. 配置文件多个监听同步
	``` BASH
	sync-dev -c sync.conf
	```
	配置文件示例：[sync.conf](https://github.com/terrywh/sync-dev/blob/master/sync.conf)

Windows 系统可考虑使用下面脚本后台启动：
```
// sync-dev.js
var sh = new ActiveXObject('Wscript.Shell');
sh.Run('sync-dev.exe -c sync.conf', 0, false)
```

使用私钥登录可靠率如下操作：
1. 本机 ssh-keygen 生成密钥；
2. 本机 ssh-copy-id user@xx.xx.xx.xx 建立远端公钥信任；

注意：
* 若在本地路径中使用相对路径，请确认其参照的目录为“当前工作路径”；
* Windows 路径分隔符在 JSON 中需要转义；

### 配置说明

``` js
// 提供用户名密码登录远程服务器
{"local_path":"D:\\data\\htdocs","remote_host":"11.22.33.44","remote_port":22,"remote_path":"/data/wuhao","remote_user":"wuhao","remote_pass":"123456"}
// 使用当前用户及其私钥验证登陆远程服务器（默认端口 22）
{"local_path":"./test1","remote_host":"11.22.33.44","remote_path":"/data/test1"}
// 使用指定用户及当前用户私钥验证登陆远程服务器（默认端口 22）
{"local_path":"D:/data/test2","remote_host":"11.22.33.44","remote_path":"/data/test2","remote_user":"wuhao"}

// local_path  - 本地同步目录（相对路径）
// remote_host - 远端服务器地址
// remote_port - 远端服务器端口，可选，默认 22
// remote_user - 帐号，可选，若不提供帐号，则使用当前用户
// remote_pass - 密码，可选，若不提供密码，则使用当前账户的私钥验证登陆 ${HOME}/.ssh/id_rsa 私钥路径
// remote_path - 远端目标目录

```

### 命令帮助

```
  -c string
        配置文件路径，使用配置文件可一次性定义多个监听同步
        （参考 sync.conf.example）
  -config string
        同 -c
  -h    命令行帮助
  -help
        同 -h
  -l string
        同 -local
  -local string
        设置监听同步的本地目录
  -r string
        同 -remote
  -remote string
        远程同步目标，例如 "wuhao:password@127.0.0.1:22/data/syncdir"
  -s    同 -sync
  -sync
        进行一次完整同步并退出（无法与 \"-config\" 同时使用）
```
