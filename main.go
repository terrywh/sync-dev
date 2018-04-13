package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	
	"github.com/terrywh/sync-dev/sync"
)

var flagHelp bool
var flagConf string
// var flagLocalPath string
// var flagRemotePath string
// var flagRemoteHost string
// var flagRemotePort uint
// var flagRemoteUser string
// var flagRemotePass string
var flagOne sync.Config
var flagSync bool
var flagLog string
func main() {
	flag.BoolVar(&flagHelp, "h", false, "命令行帮助")
	flag.BoolVar(&flagHelp, "help", false, "同 -h")
	flag.StringVar(&flagConf, "config", "", "配置文件路径，使用配置文件可一次性定义多个监听同步\n（参考 sync.conf.example）")
	flag.StringVar(&flagConf, "c", "", "同 -config")
	
	flag.StringVar(&flagOne.LocalPath, "local", "", "设置监听同步的本地目录")
	flag.StringVar(&flagOne.LocalPath, "l", "", "同 -local")
	
	flag.StringVar(&flagOne.RemotePath, "remote", "", "远程同步目标，例如 \"wuhao:password@127.0.0.1:22/data/syncdir\"")
	flag.StringVar(&flagOne.RemotePath, "r", "", "同 -remote")
	
	flag.BoolVar(&flagSync, "sync", false, "进行一次完整同步并退出（无法与 \"-config\" 同时使用）")
	flag.BoolVar(&flagSync, "s", false, "同 -sync")
	
	flag.StringVar(&flagLog, "log", "", "日志输出重定向到指定的文件")
	
	flag.Parse()
	// flagSync 
	if flagHelp || flagConf == "" && flagOne.RemotePath == "" || flagConf != "" && flagSync {
		flag.Usage()
		return
	}
	
	if flagLog != "" {
		logFile, err := os.OpenFile(flagLog, os.O_APPEND | os.O_CREATE, 0666)
		if err != nil {
			log.Fatal("[错误] 无法打开日志文件")
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	
	if flagConf != "" {
		sync.StartConf(flagConf)
		bufio.NewReader(os.Stdin).ReadBytes('\x03')
		sync.Stop()
	}else{
		parseOne()
		sync.StartOne(&flagOne)
		if flagSync {
			sync.SyncDir(flagOne.LocalPath);
			sync.Stop()
		}else{
			bufio.NewReader(os.Stdin).ReadBytes('\x03')
			sync.Stop()
		}
	}
}
func parseOne() {
	if flagOne.LocalPath == "" {
		log.Fatal("[错误] 同步本地目录未指定")
	}
	p := regexp.MustCompile(`^(([^:]+)(:([^@]+))?@)?([^:/]+)(:(\d+))?(/[^\s]+)$`)
	s := p.FindStringSubmatch(flagOne.RemotePath)
	if s == nil {
		log.Fatal("[错误] 无法解析远程同步目标")
	}
	flagOne.RemoteHost = s[5]
	if s[6] == "" {
		flagOne.RemotePort = 22
	}else{
		port, err := strconv.Atoi(s[7])
		if err != nil || flagOne.RemotePort < 1 || flagOne.RemotePort > 65535 {
			log.Fatal("[错误] 远程同步目标端口异常")
		}
		flagOne.RemotePort = uint(port)
	}
	
	flagOne.RemoteUser = s[2] // 可以为空
	flagOne.RemotePass = s[4] // 可以为空 （使用本地私钥登录）
	flagOne.RemotePath = s[8]

	if flagOne.RemotePath == "" {
		log.Fatal("[错误] 远程同步目标目录未指定")
	}
}
