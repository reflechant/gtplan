package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/reflechant/gtplan/gtp"
)

func copy(dst io.Writer, src io.Reader) (written int64, err error) {
	gtp.F()
	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, size)
	for {
		nr, er := src.Read(buf)
		log.Println(string(buf))
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func TCPtoGTP(done chan bool) {
	l, err := net.Listen("tcp", ":2000")
	if err != nil {
		log.Fatalln(err)
	}
	defer l.Close()
	conn, err := l.Accept()
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go copy(os.Stdout, conn)
	wg.Add(1)
	go copy(conn, os.Stdin)
	wg.Wait()
	done <- true
}

func GTPtoTCP(port int, done chan bool) {
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalln("Failed to dial: %v", err)
	}
	defer conn.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go copy(conn, os.Stdin)
	wg.Add(1)
	go copy(os.Stdout, conn)
	wg.Wait()
	done <- true
}

func main() {
	done := make(chan bool)
	go TCPtoGTP(done)
	go GTPtoTCP(2000, done)
	<-done
}
