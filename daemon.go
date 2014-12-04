package floki

import (
	"bitbucket.org/kardianos/osext"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var httpWg sync.WaitGroup

type gracefulListener struct {
	net.Listener
	stop    chan error
	stopped bool
}

var theListener *gracefulListener

func (gl *gracefulListener) Accept() (c net.Conn, err error) {
	c, err = gl.Listener.Accept()
	if err != nil {
		return
	}

	c = gracefulConn{Conn: c}

	httpWg.Add(1)

	return
}

func newGracefulListener(l net.Listener) (gl *gracefulListener) {
	gl = &gracefulListener{Listener: l, stop: make(chan error)}
	go func() {
		_ = <-gl.stop
		gl.stopped = true
		gl.stop <- gl.Listener.Close()
	}()
	return gl
}

func (gl *gracefulListener) Close() error {
	if gl.stopped {
		return syscall.EINVAL
	}
	gl.stop <- nil
	return <-gl.stop
}

func (gl *gracefulListener) File() *os.File {
	tl := gl.Listener.(*net.TCPListener)
	fl, _ := tl.File()
	return fl
}

type gracefulConn struct {
	net.Conn
}

func (w gracefulConn) Close() error {
	httpWg.Done()
	return w.Conn.Close()
}

func spawnChild(listener *gracefulListener) {
	file := listener.File()
	path, _ := osext.Executable()
	args := flag.Args()

	os.Setenv("FLOKI_CHILD_PROC", "1")

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{file}

	err := cmd.Start()
	if err != nil {
		log.Fatalf("gracefulRestart: Failed to launch, error: %v", err)
	}
}

var gracefulChild bool

func (f *Floki) listenHTTP(addr string, handler http.Handler, pidFile string) error {
	// in Dev & Test environments we don't need to daemonize the process
	if Env == Dev || Env == Test {
		return http.ListenAndServe(addr, handler)
	}

	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 16}

	var l net.Listener
	var err error

	if os.Getenv("FLOKI_CHILD_PROC") == "1" {
		gracefulChild = true
	}

	if gracefulChild {
		f := os.NewFile(uintptr(3), "")
		l, err = net.FileListener(f)
	} else {
		l, err = net.Listen("tcp", server.Addr)
	}

	if err != nil {
		return err
	}

	if gracefulChild {
		parent := syscall.Getppid()
		syscall.Kill(parent, syscall.SIGTERM)

		err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(syscall.Getpid())), 0660)
		if err != nil {
			log.Println("can't write pid to file:", pidFile, ". Error:", err)
		}
	}

	theListener = newGracefulListener(l)

	if !gracefulChild {
		spawnChild(theListener)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGINT, syscall.SIGTERM:
				theListener.Close()

				// waiting for running tasks to complete
				httpWg.Wait()

				f.triggerAppEvent("Shutdown")

				os.Remove(pidFile)
				os.Exit(0)

			case syscall.SIGHUP:
				f.triggerAppEvent("Reload")

				log.Println("got SIGHUP! restarting gracefully..")
				spawnChild(theListener)
			}
		}
	}()

	server.Serve(theListener)

	// waiting for running tasks to complete
	httpWg.Wait()

	return nil
}
