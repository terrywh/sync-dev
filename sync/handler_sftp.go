package sync

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"os"
	"os/user"
	
	
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
	localDir  string
}

func CreateSftpHandler(localDir, remoteDir, remoteHost, remotePort, remoteUser, remotePass string) (*SftpHandler,error) {
	// 本地目录必须存在且可访问
	localDir = filepath.Clean(filepath.FromSlash(localDir))
	info, err := os.Stat(localDir)
	if err != nil || !info.IsDir() {
		return nil, ErrBadLocalDir
	}
	remoteDir = path.Clean(remoteDir)
	if remoteUser == "" {
		user, err := user.Current()
		if err != nil {
			return nil, ErrBadRemoteUser
		}
		remoteUser = user.Username
	}
	var auth []ssh.AuthMethod
	if remotePass != "" {
		auth = []ssh.AuthMethod{
			ssh.Password(remotePass),
		}
	}else{
		homedir := os.Getenv("HOME")
		if homedir == "" {
			usr, err := user.Current()
			if err != nil {
				return nil, ErrBadRemoteUser
			}
			homedir = usr.HomeDir
		}else{
			homedir, _ = filepath.Abs(homedir)
		}
		key, err := ioutil.ReadFile(homedir + "/.ssh/id_rsa")
		if err != nil {
			return nil, err
		}
		sig, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		auth = []ssh.AuthMethod{
			ssh.PublicKeys(sig),
		}
	}
	// 尝试建立远程服务器连接
	sss, err := ssh.Dial("tcp", remoteHost + ":" + remotePort, &ssh.ClientConfig{
		User: remoteUser,
		Auth: auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, err
	}
	fff, err := sftp.NewClient(sss)
	if err != nil {
		log.Fatal(err)
	}
	info, err = fff.Stat(remoteDir)
	if err != nil || !info.IsDir() {
		return nil, ErrBadRemoteDir
	}
	return &SftpHandler{
		con: sss,
		cli: fff,
		remoteDir: remoteDir,
		localDir: localDir,
	}, nil
}

func (h *SftpHandler) Close() error {
	h.cli.Close()
	return h.con.Close()
}

func (h *SftpHandler) MapPath(lpath string) string {
	return filepath.ToSlash(h.remoteDir + lpath[len(h.localDir):])
}
// 在远端创建目录
func (h *SftpHandler) CreateDir(lpath string) error {
	return h.cli.Mkdir(h.MapPath(lpath))
}
// 删除远端目录（递归删除）
// TODO 递归转循环防止层级过深
func (h *SftpHandler) Remove(lpath string) error {
	rpath := h.MapPath(lpath)
	return h.remover(rpath)
}
func (h *SftpHandler) remover(rpath string) error {
	files, err := h.cli.ReadDir(rpath)
	
	if err == os.ErrNotExist {
		return nil
	} else if err != nil {
		// 若非目录直接删除即可
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
