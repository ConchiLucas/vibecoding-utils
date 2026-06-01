package utils

import (
	"bufio"
	"io"
	"os"
)

// LogWriter 实现 io.Writer 接口，将写入的内容逐行发送到 channel
// 同时也写入到 os.Stdout / os.Stderr 保持控制台输出
type LogWriter struct {
	ch     chan string
	mirror io.Writer // 同时写入到控制台
}

// NewLogWriter 创建一个 LogWriter，同时写入 channel 和 mirror（如 os.Stdout）
func NewLogWriter(ch chan string, mirror io.Writer) *LogWriter {
	if mirror == nil {
		mirror = os.Stdout
	}
	return &LogWriter{ch: ch, mirror: mirror}
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	// 先写入控制台
	n, err = w.mirror.Write(p)

	// 逐行发送到 channel（非阻塞，防止 channel 满时卡住命令执行）
	scanner := bufio.NewScanner(io.NopCloser(
		&bytesReader{data: p},
	))
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case w.ch <- line:
		default:
			// channel 满了就丢弃，不阻塞命令执行
		}
	}
	return n, err
}

// bytesReader 封装 []byte 为 io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
