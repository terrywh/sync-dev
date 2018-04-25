package sync

import (
	"errors"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"os"
	"os/user"
	"time"
	
	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
)

var (
	ErrBadRemoteHost = errors.New("远端服务器无法连接或登陆失败")
	ErrBadRemoteDir = errors.New("远端路径不存在或无法访问")
	ErrBadRemoteUser = errors.New("远端服务器帐号未提供或公钥无法获取")
)

type SftpHandler struct {
	con *ssh.Client
	cli *sftp.Client
	remoteDir string
	remoteHost string
	remotePort string
	remoteUser string
	remotePass string
	localDir  string
}

func CreateSftpHandler(localDir, remoteDir, remoteHost, remotePort, remoteUser, remotePass string) *SftpHandler {
	var h SftpHandler
	h.localDir = filepath.Clean(filepath.FromSlash(localDir))
	h.remoteDir = path.Clean(remoteDir)
	h.remoteHost = remoteHost
	h.remotePort = remotePort
	if h.remotePort == "" {
		h.remotePort = "22"
	}
	h.remoteUser = remoteUser
	if h.remoteUser == "" {
		user, _ := user.Current()
		h.remoteUser = user.Username
	}
	h.remotePass = remotePass
	return &h
}

func (h *SftpHandler) Dial() error {
	// 本地目录必须存在且可访问
	info, err := os.Stat(h.localDir)
	if err != nil || !info.IsDir() {
		return ErrBadLocalDir
	}
	if h.remoteUser == "" {
		return ErrBadRemoteUser
	}
	var auth []ssh.AuthMethod
	if h.remotePass != "" {
		auth = []ssh.AuthMethod{
			ssh.Password(h.remotePass),
		}
	}else{
		homedir := os.Getenv("HOME")
		if homedir == "" {
			usr, err := user.Current()
			if err != nil {
				return ErrBadRemoteUser
			}
			homedir = usr.HomeDir
		}else{
			homedir, _ = filepath.Abs(homedir)
		}
		key, err := ioutil.ReadFile(homedir + "/.ssh/id_rsa")
		if err != nil {
			return err
		}
		sig, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return err
		}
		auth = []ssh.AuthMethod{
			ssh.PublicKeys(sig),
		}
	}
	// 尝试建立远程服务器连接
	sss, err := ssh.Dial("tcp", h.remoteHost + ":" + h.remotePort, &ssh.ClientConfig{
		User: h.remoteUser,
		Auth: auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 3 * time.Second,
	})
	if err != nil {
		return err
	}
	fff, err := sftp.NewClient(sss)
	if err != nil {
		return err
	}
	info, err = fff.Stat(h.remoteDir)
	if err != nil || !info.IsDir() {
		return ErrBadRemoteDir
	}
	h.con = sss
	h.cli = fff
	return nil
}

func (h *SftpHandler) Close() error {
	h.cli.Close()
	return h.con.Close()
}

func (h *SftpHandler) IsConnectionFailed(err error) bool {
	switch err.Error() {
	case "failed to send packet header: EOF":
		return true
	}
	return false
}

func (h *SftpHandler) MapPath(lpath string) string {
	return filepath.ToSlash(h.remoteDir + lpath[len(h.localDir):])
}
// 在远端创建目录
func (h *SftpHandler) CreateDir(lpath string) error {
	return h.cli.Mkdir(h.MapPath(lpath))
}
func (h *SftpHandler) CreateFile(lpath string) error {
	_, err := h.cli.Create(h.MapPath(lpath))
	return err
}
// 删除远端目录（递归删除）
// TODO 递归转循环防止层级过深
func (h *SftpHandler) Remove(lpath string) error {
	rpath := h.MapPath(lpath)
	return h.remover(rpath)
}
func (h *SftpHandler) remover(rpath string) error {
	files, err := h.cli.ReadDir(rpath)
	
	if err != nil {
		// 非目录 或不存在 直接删除即可
		return h.cli.Remove(rpath)
	}
	for _, info := range files {
		if info.IsDir() {
			h.remover(rpath + "/" + info.Name())
		} else {
			h.cli.Remove(rpath + "/" + info.Name())
		}
	}
	return h.cli.RemoveDirectory(rpath)
}
// 将本地文件传输到远端
func (h *SftpHandler) UploadFile(lpath string, lfile *os.File) error {
	rpath := h.MapPath(lpath)
	rfile, err := h.cli.Create(rpath)
	if err != nil {
		return err
	}
	defer rfile.Close()
	lfile.Seek(0, 0)
	_, err = io.Copy(rfile, lfile)
	return err
}
// 重命名
func (h *SftpHandler) Rename(opath, npath string) error {
	return h.cli.Rename(h.MapPath(opath), h.MapPath(npath))
}
// 目录完整同步(将本地文件全部上传，本地不存在的远端文件不受影响)
func (h *SftpHandler) SyncDir(lpath string) error {
	lfile, err := os.Open(lpath)
	if err != nil {
		return err
	}
	defer lfile.Close()
	lstat, err := lfile.Stat()
	if err != nil {
		return err
	}
	rpath := h.MapPath(lpath)
	rstat, err := h.cli.Stat(rpath)
	
	if lstat.IsDir() {
		if err != nil {
			h.CreateDir(lpath)
		}else if !rstat.IsDir() {
			h.cli.Remove(lpath)
			h.CreateDir(lpath)
		}
		// 同步目录中的所有内容
		names, err := lfile.Readdirnames(0)
		if err != nil {
			return err
		}
		for _, name := range names {
			h.SyncDir(lpath + "/" + name)
		}
	}else{
		if err != nil || !rstat.IsDir() {
			h.UploadFile(lpath, lfile)
		}else{ // 远端是个目录
			h.Remove(lpath)
			h.UploadFile(lpath, lfile)
		}
	}
	return nil
}
