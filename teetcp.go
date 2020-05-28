package main

import (
	"flag"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	reuseport "github.com/kavu/go_reuseport"
	"github.com/smallnest/log"
)

var (
	addr      = flag.String("addr", ":8972", "listen address by teetcp")
	adminAddr = flag.String("admin", ":9981", "admin address")
	s1        = flag.String("s1", ":6379", "server1 address")
	s2        = flag.String("s2", "", "server1 address")
	logDir    = flag.String("logdir", "", "log directory")
)

func main() {
	log.SetLevel(log.LOG_LEVEL_INFO)
	log.SetHighlighting(false)

	// 输出日志到文件
	if *logDir != "" {
		log.SetRotateByDay()
		os.MkdirAll(*logDir, 0777)
		log.SetOutputByName(filepath.Join(*logDir, "teedis.log"))
	}

	ln, err := reuseport.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	ts := New(ln, *s1, *s2)
	if *s2 != "" {
		close(ts.s2ch)
	}

	go ts.startAdminServer(*adminAddr)

	ts.startServer()
}

type TCPServer struct {
	ln   net.Listener
	s1   string
	s2   string
	s2ch chan struct{}
	done chan struct{}
}

func New(ln net.Listener, s1, s2 string) *TCPServer {
	return &TCPServer{
		ln:   ln,
		s1:   s1,
		s2:   s2,
		s2ch: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (s *TCPServer) startServer() {
	for {
		select {
		case <-s.done:
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				return
			}
			go s.handleConn(conn)
		}
	}
}

func (s *TCPServer) handleConn(conn net.Conn) {
	conn.(*net.TCPConn).SetKeepAlivePeriod(time.Minute)

	s1, err := net.DialTimeout("tcp", s.s1, 10*time.Second)
	if err != nil {
		log.Fatalf("failed to dial s1 %s: %v", s.s1, err)
	}
	s1.(*net.TCPConn).SetKeepAlivePeriod(time.Minute)

	tr := TeeReader(conn, nil)

	connClose := make(chan struct{})

	go s.startWritebackup(connClose, tr)
	go transfer(conn, s1)
	go transferCallback(s1, tr, func() { close(connClose) })
}

// start double write
func (s *TCPServer) startWritebackup(connClose chan struct{}, tr *teeReader) {
	select {
	case <-connClose:
		return
	case <-s.done:
		return
	case <-s.s2ch:
		if s.s2 != "" { // 设置了双写地址，开始双写
			s2, err := net.DialTimeout("tcp", s.s2, 10*time.Second)
			if err != nil {
				log.Errorf("failed to dial s2 %s: %v", s.s1, err)
			}
			s2.(*net.TCPConn).SetKeepAlivePeriod(time.Minute)
			tr.w = s2
			go io.Copy(ioutil.Discard, s2)

		}
	}
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func transferCallback(dst io.WriteCloser, src io.ReadCloser, closeCallback func()) {
	defer dst.Close()
	defer src.Close()
	if closeCallback != nil {
		defer closeCallback()
	}

	io.Copy(dst, src)
}
