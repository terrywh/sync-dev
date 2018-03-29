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
	h Handler
	c chan notify.EventInfo
}
// Create 生成一个新的 Watcher 对象
func CreateWatcher(dir string, h Handler) (*Watcher,error) {
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
		if ec.Event() == el.Event() && ec.Event() == notify.Rename {
			w.handle(ec.Event(), el.Path(), ec.Path())
		}else if ec.Event() != el.Event() || ec.Path() != el.Path() {
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
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("failed to open local file:", path)
	}
	defer file.Close()
	stat, err := file.Stat()
	if stat == nil {
		
	}else if stat.IsDir() {
		switch ev {
		case notify.Rename:
			w.h.Rename(oldPath, path)
		case notify.Create:
			w.h.CreateDir(path)
		case notify.Remove:
			w.h.RemoveDir(path)
		}
	}else{
		switch ev {
		case notify.Create:
		case notify.Write:
			w.h.UploadFile(path, file)
		case notify.Rename:
			w.h.Rename(oldPath, path)
		case notify.Remove:
			w.h.RemoveFile(path)
		}
	}
}
