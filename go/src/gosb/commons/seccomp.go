package commons

import (
	"strconv"
	"strings"
	"syscall"
)

const (
	DELIMITER_SYSCLASS = ","
	BITMASK_LEN        = (syscall.SYS_PRLIMIT64)/64 + 1
)

type SyscallMask = [BITMASK_LEN]uint64

var SyscallClasses = map[string][]int{
	"mem": []int{syscall.SYS_MMAP, syscall.SYS_MPROTECT, syscall.SYS_BRK},
	"file": []int{
		syscall.SYS_CLOSE, syscall.SYS_CREAT, syscall.SYS_OPEN, syscall.SYS_OPENAT,
		/*syscall.SYS_NAME_TO_HANDLE_AT, syscall.SYS_OPEN_BY_HANDLE_AT, syscall.SYS_MEMFD_CREATE,*/
		syscall.SYS_MKNOD, syscall.SYS_MKNODAT, syscall.SYS_RENAME, syscall.SYS_RENAMEAT,
		/*syscall.SYS_RENAMEAT2,*/ syscall.SYS_TRUNCATE, syscall.SYS_FTRUNCATE, syscall.SYS_FALLOCATE,
		syscall.SYS_MKDIR, syscall.SYS_MKDIRAT, syscall.SYS_RMDIR, syscall.SYS_GETCWD, syscall.SYS_CHDIR,
		syscall.SYS_FCHDIR, syscall.SYS_CHROOT, syscall.SYS_CHROOT, syscall.SYS_GETDENTS,
		syscall.SYS_GETDENTS64, syscall.SYS_LOOKUP_DCOOKIE, syscall.SYS_LINK, syscall.SYS_LINKAT,
		syscall.SYS_SYMLINK, syscall.SYS_SYMLINKAT, syscall.SYS_UNLINK, syscall.SYS_UNLINKAT,
		syscall.SYS_READLINK, syscall.SYS_READLINKAT, syscall.SYS_UMASK, syscall.SYS_STAT,
		syscall.SYS_LSTAT, syscall.SYS_FSTAT /*syscall.SYS_FSTATAT,*/, syscall.SYS_CHMOD, syscall.SYS_FCHMOD,
		syscall.SYS_FCHMODAT, syscall.SYS_CHOWN, syscall.SYS_LCHOWN, syscall.SYS_FCHOWNAT,
		syscall.SYS_UTIME, syscall.SYS_UTIMES, syscall.SYS_FUTIMESAT, syscall.SYS_UTIMENSAT,
		syscall.SYS_ACCESS, syscall.SYS_FACCESSAT, syscall.SYS_IOCTL, syscall.SYS_FCNTL, syscall.SYS_DUP,
		syscall.SYS_DUP2, syscall.SYS_DUP3, syscall.SYS_FLOCK, syscall.SYS_READ, syscall.SYS_READV,
		/*syscall.SYS_PREAD,*/ syscall.SYS_PREADV, syscall.SYS_WRITE, syscall.SYS_WRITEV, /*syscall.SYS_PWRITE,*/
		syscall.SYS_PWRITEV, syscall.SYS_LSEEK, syscall.SYS_SENDFILE, syscall.SYS_FDATASYNC, syscall.SYS_FSYNC,
		syscall.SYS_MSYNC, syscall.SYS_SYNC_FILE_RANGE, syscall.SYS_SYNC, /*syscall.SYS_SYNCFS,*/
		syscall.SYS_IO_SETUP, syscall.SYS_IO_DESTROY, syscall.SYS_IO_SUBMIT, syscall.SYS_IO_CANCEL,
		syscall.SYS_IO_GETEVENTS, syscall.SYS_SELECT, syscall.SYS_PSELECT6, syscall.SYS_POLL,
		syscall.SYS_PPOLL, syscall.SYS_EPOLL_CREATE, syscall.SYS_EPOLL_CREATE1, syscall.SYS_EPOLL_CTL,
		syscall.SYS_EPOLL_WAIT, syscall.SYS_EPOLL_PWAIT, syscall.SYS_INOTIFY_INIT, syscall.SYS_INOTIFY_INIT1,
		syscall.SYS_INOTIFY_ADD_WATCH, syscall.SYS_INOTIFY_RM_WATCH, syscall.SYS_FANOTIFY_INIT,
		syscall.SYS_FANOTIFY_MARK, syscall.SYS_FADVISE64, syscall.SYS_READAHEAD, /*syscall.SYS_GETRANDOM,*/
	},
	"net": []int{
		syscall.SYS_SOCKET, syscall.SYS_SOCKETPAIR, syscall.SYS_SETSOCKOPT,
		syscall.SYS_GETSOCKOPT, syscall.SYS_GETSOCKNAME, syscall.SYS_GETPEERNAME,
		syscall.SYS_BIND, syscall.SYS_LISTEN, syscall.SYS_ACCEPT, syscall.SYS_ACCEPT4,
		syscall.SYS_CONNECT, syscall.SYS_SHUTDOWN, syscall.SYS_RECVFROM, syscall.SYS_RECVMSG,
		syscall.SYS_SENDTO, syscall.SYS_SENDMSG, syscall.SYS_SETHOSTNAME, syscall.SYS_SETDOMAINNAME},
	"io":      []int{syscall.SYS_WRITE, syscall.SYS_READ},
	"default": []int{syscall.SYS_EXIT_GROUP},
}

var SyscallConfigs map[string]SyscallMask = nil
var SyscallAll = SyscallMask{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}
var syscallNone = SyscallMask{0, 0, 0, 0, 0}

func init() {
	SyscallConfigs = make(map[string]SyscallMask)
	for k, v := range SyscallClasses {
		mask := transform(v)
		SyscallConfigs[k] = mask
	}
}

func transform(values []int) SyscallMask {
	var mask SyscallMask
	for _, v := range values {
		idx, idy := SysCoords(v)
		Check(idx < BITMASK_LEN)
		mask[idx] |= 1 << idy
	}
	return mask
}

//go:nosplit
func SysCoords(nb int) (int, int) {
	return nb / 64, nb % 64
}

func ParseSyscalls(config string) (SyscallMask, error) {
	Check(SyscallConfigs != nil)
	conf, err := strconv.Unquote(config)
	if err != nil {
		conf = config
	}
	if len(conf) == 0 {
		return SyscallAll, nil
	}
	var mask SyscallMask
	entries := strings.Split(conf, DELIMITER_SYSCLASS)
	// Add the default entries
	entries = append(entries, "default")
	for _, v := range entries {
		e, ok := SyscallConfigs[v]
		Check(ok)
		Add(&mask, e)
	}
	return mask, nil
}

func Add(m *SyscallMask, other SyscallMask) {
	Check(len(m) == len(other))
	for i := range m {
		m[i] |= other[i]
	}
}
