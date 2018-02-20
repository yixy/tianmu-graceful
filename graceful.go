package graceful

import (
	"net"
	"net/http"
	"os/exec"
	"os"
	"flag"
	"context"
	"time"
	"syscall"
	"os/signal"
	"strconv"
	"errors"
)

var server *http.Server
var f *os.File
var pid int = 0
var worker = flag.Bool("worker", false, "for creating worker process.")
var workerExit chan error // channel waiting for worker.Wait()

func StartServer(s *http.Server)error{
	defer func(){
		if pid!=0{
			signalOperation(pid,"SIGINT")
		}
	}()
	flag.Parse()
	server=s
	if *worker{
		//初始化worker
		f := os.NewFile(3, "")
		listener, err := net.FileListener(f)
		if err!=nil{
			return errors.New("create worker listener error:"+err.Error())
		}

		go func(){
			err = server.Serve(listener)
			if err!=nil{
				panic(err)
			}
		}()

		return workerSignalHandler()
	}else {
		//初始化Master
		a, err := net.ResolveTCPAddr("tcp", server.Addr)
		if err != nil {
			return errors.New("master ResolveTCPAddr error:"+err.Error())
		}
		l, err := net.ListenTCP("tcp", a)
		if err != nil {
			return errors.New("master ListenTCP error:"+err.Error())
		}
		f, err = l.File()
		if err != nil {
			return errors.New("master failed to retreive fd:"+err.Error())
		}
		if err := l.Close(); err != nil {
			return errors.New("master failed to close listener:"+err.Error())
		}

		pid,err=initWorker()
		if err!=nil{
			return errors.New("initWorker err:"+err.Error())
		}
		return masterSignalHandler()
	}
}

//创建worker进程
func initWorker()(int,error){
	args := []string{"-worker"}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// put socket FD at the first entry
	cmd.ExtraFiles = []*os.File{f}
	err:=cmd.Start()
	go func() {
		workerExit <- cmd.Wait()
	}()
	return cmd.Process.Pid,err
}

//worker监听信号
func workerSignalHandler() error{
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		// timeout context for shutdown
		ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2:
			// stop
			signal.Stop(ch)
			return server.Shutdown(ctx)
		}
	}
}

//master监听信号
func masterSignalHandler()error{
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			// stop
			signal.Stop(ch)
			return signalOperation(pid,"SIGINT")
		case syscall.SIGUSR2:
			// reload
			pidNew,err:=initWorker()
			if err != nil {
				return err
			}
			time.Sleep(time.Duration(10000)*time.Millisecond)
			err=signalOperation(pid,"SIGINT")
			if err!=nil{return err}
			pid=pidNew
		}
	}
}

//kill信号操作
func signalOperation(processId int,sig string)error{
	args := []string{"-s",sig,strconv.Itoa(processId)}
	cmd := exec.Command("kill", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}