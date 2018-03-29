package sync

import (
	"encoding/json"
	"path/filepath"
	"fmt"
	"io"
	"log"
	"os"
)

type Config struct {
	LocalPath  string `json:"local_path"`
	RemoteHost string `json:"remote_host"`
	RemotePort uint  `json:"remote_port"`
	RemotePath string `json:"remote_path"`
	RemoteUser string `json:"remote_user"`
	RemotePass string `json:"remote_pass"`
}

type Sync struct {
	h  Handler
	w *Watcher
	c *Config
}

var syncs []Sync

func StartConf(cfile string) {
	syncs = make([]Sync, 0, 4)
	log.Println("读取同步配置文件:", cfile)
	file, err := os.Open(cfile)
	if err != nil {
		log.Fatal("[错误] 同步配置文件无法打开")
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	for {
		var c Config
		if err := dec.Decode(&c); err == io.EOF {
			break
		}else if err != nil {
			log.Fatal("[错误] 解析同步配置出现错误: ", err)
		}
		appendSync(&c)
	}
	startSyncs()
}
func StartOne(c *Config) {
	appendSync(c)
	startSyncs()
}
func appendSync(c *Config) {
	if c.RemotePort < 1 || c.RemotePort > 65536 {
		c.RemotePort = 22
	}
	var s Sync
	s.c = c
	
	log.Printf("远程目标: %s:%d%s", s.c.RemoteHost, s.c.RemotePort, s.c.RemotePath)
	h, err := CreateSftpHandler(s.c.LocalPath, s.c.RemotePath, s.c.RemoteHost, fmt.Sprintf("%d", s.c.RemotePort), s.c.RemoteUser, s.c.RemotePass)
	if err != nil {
		log.Fatal("[错误] 无法创建 SFTP 处理句柄: ", err)
	}
	s.h = h
	log.Println("本地路径:", s.c.LocalPath)
	w, err := CreateWatcher(s.c.LocalPath, h)
	if err != nil {
		s.h.Close()
		log.Fatal("[错误] 无法创建本地文件监听: ", err)
	}
	s.w = w
	syncs = append(syncs, s)
}
func startSyncs() {
	log.Println("启动服务:")
	var i int
	for i=0 ;i<len(syncs); i++ {
		log.Printf("(%d) %s => %s:%d%s", i+1, syncs[i].c.LocalPath, syncs[i].c.RemoteHost, syncs[i].c.RemotePort, syncs[i].c.RemotePath)
		go syncs[i].w.Watch()
	}
}
func Stop() {
	log.Println("关闭服务:")
	for i:=0 ;i<len(syncs); i++ {
		log.Println(fmt.Sprintf("(%d)", i+1), syncs[i].c.LocalPath)
		syncs[i].w.Close()
		syncs[i].h.Close()
	}
}
func SyncDir(dir string) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal("[错误] 无法确认指定同步目录: ", err)
	}
	log.Println("(1) 完整同步 ...")
	syncs[0].h.SyncDir(dir)
	log.Printf("(%d) 同步完毕", 1)
}
