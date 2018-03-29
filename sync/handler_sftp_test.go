package sync

import (
	"os"
	"testing"
)

var handler *SftpHandler
var err   error
var lroot = "D:\\data\\htdocs\\src\\github.com\\terrywh\\node-toolkit\\test"
var rroot = "/data/wuhao/test"

func TestRemoteConnection(t *testing.T) {
	handler, err = CreateSftpHandler(lroot + "\\", rroot + "/", "11.22.33.44", "22", "wuhao", "123456")
	if err != nil {
		t.Fatal("failed to connecto remote host")
	}
	if handler.localDir != lroot || handler.remoteDir != rroot {
		t.Fatal("failed to normalize path: ", handler.localDir, " <=> ", handler.remoteDir)
	}
}

func TestPathMapping(t *testing.T) {
	if handler.MapPath(lroot + "\\this is the name") != rroot + "/this is the name" {
		t.Fatal("failed to map local path to remote path")
	}
}

func SameFile(f1, f2 os.FileInfo) bool {
	return f1.Name() == f2.Name() && f1.Size() == f2.Size()
}

func TestRemoveActions(t *testing.T) {
	var err error
	var ri, li os.FileInfo
	handler.CreateDir(lroot + "/inner1")
	ri, err  = handler.cli.Stat(rroot + "/inner1")
	if err != nil {
		t.Fatal("failed to create dir", err)
	}
	if !ri.IsDir() {
		t.Fatal("failed to create dir: not a directory")
	}
	file, err := os.Open(lroot + "/a.txt")
	if err != nil {
		t.Fatal("failed to upload file: local file does not exist")
	}
	handler.UploadFile(lroot + "/a.txt", file)
	li, _ = os.Stat(lroot + "/a.txt")
	ri, _ = handler.cli.Stat(rroot + "/a.txt")
	if li == nil || ri == nil || !SameFile(li, ri) {
		t.Fatal("failed to upload file: not the same")
	}
	li, _ = os.Stat(lroot + "/a.txt")
	ri, _ = handler.cli.Stat(rroot + "/a.txt")
	if li == nil || ri == nil || !SameFile(li, ri) {
		t.Fatal("failed to upload file: not the same")
	}
	
	file, err = os.Open(lroot + "/inner/b.txt")
	if err != nil {
		t.Fatal("failed to upload file: local file does not exist")
	}
	handler.UploadFile(lroot + "/inner/b.txt", file)
	
	
	handler.Rename(lroot + "/a.txt", lroot + "/b.txt")
	if err != nil {
		t.Fatal("failed to rename file", err)
	}
	ri, _  = handler.cli.Stat(rroot + "/b.txt")
	if ri == nil {
		t.Fatal("failed to rename file: target file not exist")
	}
	
	handler.Rename(lroot + "/inner1", lroot + "/inner2")
	ri, _  = handler.cli.Stat(rroot + "/inner2")
	if ri == nil || !ri.IsDir() {
		t.Fatal("failed to rename dir: target dir not exist")
	}
	
	handler.RemoveFile(lroot + "/b.txt")
	ri, _  = handler.cli.Stat(rroot + "/b.txt")
	if ri != nil {
		t.Fatal("failed to remove file: target file still exists")
	}
	
	handler.RemoveDir(lroot + "/inner2")
	ri, _  = handler.cli.Stat(rroot + "/inner2")
	if ri != nil {
		t.Fatal("failed to remove dir: target dir still exists")
	}
}
