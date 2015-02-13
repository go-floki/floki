package floki

import (
	"bitbucket.org/kardianos/osext"
	"flag"
	"github.com/facebookgo/grace/gracehttp"
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
var executablePath string

type gracefulListener struct {
	net.Listener
	stop    chan error
	stopped bool
}

var theListener *gracefulListener

func (gl *gracefulListener) Accept() (c net.Conn, err error) {
	c, err = gl.Listener.Accept()
	if err != nil {
		log.Println("error accepting connection:", err)
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
	args := flag.Args()

	os.Setenv("FLOKI_CHILD_PROC", "1")

	cmd := exec.Command(executablePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{file}

	err := cmd.Start()
	if err != nil {
		log.Fatalf("gracefulRestart: Failed to launch binary '%s', error: %v", executablePath, err)
	}
}

var gracefulChild bool

func (f *Floki) listenHTTP(addr string, handler http.Handler, pidFile string) error {
	go func() {
		// wait for parent process to delete it's pid file so it won't delete ours
		time.Sleep(time.Second * 1)

		err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(syscall.Getpid())), 0660)
		if err != nil {
			log.Println("can't write pid to file:", pidFile, ". Error:", err)
		}
	}()

	if true {
		return gracehttp.Serve(
			&http.Server{Addr: addr, Handler: handler},
		)
	}

	// in Dev & Test environments we don't need to daemonize the process
	/*
		if Env == Dev || Env == Test {
			return http.ListenAndServe(addr, handler)
		} */

	log := f.Logger()

	executablePath, _ = osext.Executable()

	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
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
		log.Println("killing parent:", parent)
		syscall.Kill(parent, syscall.SIGTERM)

		go func() {
			// wait for parent process to delete it's pid file so it won't delete ours
			time.Sleep(time.Second * 1)

			err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(syscall.Getpid())), 0660)
			if err != nil {
				log.Println("can't write pid to file:", pidFile, ". Error:", err)
			}
		}()
	}

	theListener = newGracefulListener(l)

	if !gracefulChild {
		spawnChild(theListener)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for s := range c {
			log.Println("got signal:", s)

			switch s {
			case syscall.SIGTERM:
				os.Remove(pidFile)
				theListener.Close()

				// waiting for running tasks to complete
				httpWg.Wait()

				f.triggerAppEvent("Shutdown")

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
