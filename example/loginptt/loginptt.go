package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kinabcd/govterm"

	"golang.org/x/crypto/ssh"
)

type Target interface {
	Action(string, io.Writer) bool
}

type TargetLogin struct {
	Account  string
	Password string
}

func (t *TargetLogin) Action(display string, w io.Writer) bool {
	if strings.Contains(display, "請輸入代號") {
		w.Write([]byte(t.Account))
		w.Write([]byte("\r"))
		w.Write([]byte(t.Password))
		w.Write([]byte("\r"))
	} else if strings.Contains(display, "按任意鍵繼續") {
		w.Write([]byte(" "))
	} else if strings.Contains(display, "您想刪除其他重複登入的連線嗎") {
		w.Write([]byte("n\r"))
	} else if strings.Contains(display, "【主功能表】") && strings.Contains(display, "人, 我是") {
		return true
	} else if strings.Contains(display, "【分類看板】") || strings.Contains(display, "看板列表") {
		w.Write([]byte("\x1b[D\x1b[D\x1b[D"))
	}
	return false
}

type TargetUser struct{ Account string }

func (t *TargetUser) Action(display string, w io.Writer) bool {
	if strings.Contains(display, "【主功能表】") && strings.Contains(display, "人, 我是") {
		w.Write([]byte("t\rq\r"))
		time.Sleep(2 * time.Second)
	} else if strings.Contains(display, "查詢網友") && strings.Contains(display, "請輸入使用者代號") {
		w.Write([]byte(t.Account))
		w.Write([]byte("\r"))
	} else if strings.Contains(display, "《登入次數》") && strings.Contains(display, " 次 ") {
		w.Write([]byte("\x1b[D\x1b[D\x1b[D\x1b[D\x1b[D"))
		return true
	} else if strings.Contains(display, "按任意鍵繼續") {
		w.Write([]byte(" "))
	}
	return false
}

type TargetLogout struct{}

func (t *TargetLogout) Action(display string, w io.Writer) bool {
	if strings.Contains(display, "按任意鍵繼續") {
		w.Write([]byte(" "))
		if strings.Contains(display, "此次停留時間") {
			return true
		}
	} else if strings.Contains(display, "【主功能表】") && strings.Contains(display, "人, 我是") {
		w.Write([]byte("g\r"))
	} else if strings.Contains(display, "您確定要離開【 批踢踢實業坊 】嗎") {
		w.Write([]byte("y\r"))
	} else {
		w.Write([]byte("\x1b[D\x1b[D\x1b[D\x1b[D\x1b[D"))
	}
	return false
}

func main() {
	account := ""
	password := ""
	flag.Func("a", "Account", func(s string) error { account = s; return nil })
	flag.Func("p", "Password", func(s string) error { password = s; return nil })
	flag.Parse()
	ctx := context.Background()
	dialer := &net.Dialer{}
	logger := slog.Default().WithGroup("PTT")
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", "ptt.cc:22")
	if err != nil {
		logger.Error("dial to ptt.cc failed", slog.String("err", err.Error()))
		return
	}
	defer conn.Close()
	// 建立SSH使用者端連線
	sshConn, ch, req, err := ssh.NewClientConn(conn, "ptt.cc:22", &ssh.ClientConfig{
		User: "bbsu",

		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		logger.Error("SSH dial error", slog.String("err", err.Error()))
		return
	}
	defer sshConn.Close()
	client := ssh.NewClient(sshConn, ch, req)

	// 建立新對談
	session, err := client.NewSession()
	if err != nil {
		logger.Error("new session error", slog.String("err", err.Error()))
		return
	}
	defer session.Close()

	w, _ := session.StdinPipe()
	out, _ := session.StdoutPipe()
	reader2 := bufio.NewReader(out)
	session.Stderr = os.Stderr // 對談錯誤輸出關聯到系統標準錯誤輸出裝置
	requestEnd := false
	str := ""
	screen := govterm.NewScreen(80, 24)
	stream := govterm.NewStream(screen)

	waitTimer := time.NewTimer(1 * time.Second)
	set := false

	go func() {
		var targets []Target = []Target{
			&TargetLogin{Account: account, Password: password},
			&TargetUser{Account: account},
			&TargetLogout{},
		}
		var state = 0
		for {
			<-waitTimer.C
			set = false
			d := screen.Display()
			for i, s := range d {
				fmt.Printf("%02d %s %02d\n", i+1, s, i+1)
			}
			display := strings.Join(d, "\n")
			target := targets[state]
			if target == nil {
				sshConn.Close()
				return
			}
			if target.Action(display, w) {
				state += 1
				waitTimer.Reset(2 * time.Second)
			}
		}
	}()
	go func() {
		for {
			l, _, err := reader2.ReadRune()
			if err != nil {
				return
			}
			stream.WriteString(string(l))
			if !set {
				waitTimer.Reset(1 * time.Second)
			}
		}
	}()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回顯（0禁用，1啟動）
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, //output speed = 14.4kbaud
	}
	if err = session.RequestPty("xterm", 24, 80, modes); err != nil {
		logger.Error("request pty error", slog.String("err", err.Error()))
		return
	}
	if err = session.Shell(); err != nil {
		logger.Error("start shell error", slog.String("err", err.Error()))
		return
	}
	done := make(chan error, 5)
	go func() {
		if err = session.Wait(); err != nil {
			if requestEnd && errors.Is(err, &ssh.ExitMissingError{}) {
				done <- nil
			} else {
				done <- err
			}
		}
	}()
	select {
	case <-ctx.Done():
		logger.Warn("Canceled", slog.String("str", str))
	case err := <-done:
		if err != nil {
			logger.Warn("Error", slog.String("error", err.Error()))
		}
	}
}
