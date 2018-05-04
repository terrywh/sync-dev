package sync

import (
	"container/list"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
)

var (
	ErrBadLocalDir = errors.New("本地路径不存在或无法访问")
)
// Watcher 用于监听指定路径的变更信息
type Watcher struct {
	path  string
	cache *list.List
	thres *time.Timer
	h     *SftpHandler
	c chan notify.EventInfo
}
// Create 生成一个新的 Watcher 对象
func CreateWatcher(dir string, h *SftpHandler) (*Watcher,error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, ErrBadLocalDir
	}
	w := Watcher{
		path:  dir,
		cache: list.New(),
		h:     h,
		c:     make(chan notify.EventInfo, 8),
	}
	return &w, (&w).start()
}
// Stop 用于停止当前 Watcher 监听过程
func (w *Watcher) Close() {
	notify.Stop(w.c)
	w.thres.Stop()
	close(w.c)
}

// Start 启动监听（发生变更时发出消息）
func (w *Watcher) start() error {
	// 递归监听指定路径
	err := notify.Watch(w.path+"\\...", w.c, notify.All)
	if err != nil {
		return err
	}
	return nil
}

func (w *Watcher) Watch() {
	d := 100 * time.Millisecond
	w.thres = time.NewTimer(d)
	w.thres.Stop()
	for {
		select {
		case <-w.thres.C:
			w.coalesce()
		case e, ok := <-w.c:
			if !ok {
				break
			}
			w.cache.PushBack(e)
			w.thres.Reset(d)
			log.Println(e)
		}
	}
}

func (w *Watcher) coalesce() {
	for i := w.cache.Front(); i != nil; i = i.Next() {
		ec := i.Value.(notify.EventInfo)
		if i.Prev() == nil { // 没有前置事件
			if ec.Event() != notify.Rename {
				w.handle(ec.Event(), ec.Path(), "")
			}
			continue
		}
		el := i.Prev().Value.(notify.EventInfo) // 前置事件不同
		// 特殊处理 重命名消息
		if el.Event() == notify.Rename && ec.Event() == notify.Rename {
			w.handle(ec.Event(), el.Path(), ec.Path())
		}else if ec.Event() == notify.Write && el.Event() == notify.Create &&
			ec.Path() == el.Path() {
			// 连续的 create  + write 则 write 可以不处理
		}else if ec.Event() != notify.Rename && (ec.Event() != el.Event() || ec.Path() != el.Path()) {
			w.handle(ec.Event(), ec.Path(), "")
		}else{
			
		}
	}
	// 清理
	var n *list.Element
	for i := w.cache.Front(); i != nil; i = n {
		n = i.Next()
		w.cache.Remove(i)
		// Remove 后 i.Next() = nil
	}
}

func (w *Watcher) handle(ev notify.Event, path, oldPath string) {
	var err error
	if ev == notify.Remove {
		err = w.h.Remove(path)
		w.after(err)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		log.Println("[警告] 无法访问本地文件或目录:", path)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || stat == nil {
		return
	}else if stat.IsDir() {
		switch ev {
		case notify.Rename:
			err = w.h.Rename(oldPath, path)
		case notify.Create:
			err = w.h.CreateDir(path)
			// 移动文件夹时需要同步内容
			names, _ := file.Readdirnames(1)
			if len(names) > 0 {
				w.h.SyncDir(path)
			}
		}
	}else{
		switch ev {
		case notify.Create:
			fallthrough
		case notify.Write:
			err = w.h.UploadFile(path, file)
		case notify.Rename:
			err = w.h.Rename(oldPath, path)
		}
	}
	w.after(err)
}
func (w *Watcher) after(err error) {
	if err != nil && w.h.IsConnectionFailed(err) {
		i := 0
		for {
			i++;
			log.Printf("[警告] 远程连接异常，正在尝试 (%d) ...\n", i)
			w.h.Close()
			err = w.h.Dial()
			if err == nil {
				break
			}
			time.Sleep(time.Duration(2 + i*2) * time.Second)
		}
		log.Println("[警告] 连接已恢复.")
	}
}
