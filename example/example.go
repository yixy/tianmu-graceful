package main

import (
	"net/http"
	"fmt"
	"html"
	"time"
	graceful "github.com/yixy/tianmu-graceful"
	"os"
	"strings"
)

func main(){
	serveMux:=http.NewServeMux()
	serveMux.HandleFunc("/example",helloHandle)

	server := &http.Server{
		Addr: ":9999",
		Handler:serveMux,
		ReadTimeout:time.Duration(20000)*time.Millisecond,
		WriteTimeout:time.Duration(20000)*time.Millisecond,
	}
	err:=graceful.StartServer(server)
	if err!=nil{
		panic(err)
	}
}

func helloHandle(w http.ResponseWriter, r *http.Request) {
	params:=""
	for _,item:=  range strings.Split(r.URL.RawQuery,"&"){
		value:=strings.Split(item,"=")
		params+=value[1]
	}
	fmt.Fprintf(w, "Hello, %q,%d,%s\n", html.EscapeString(r.URL.Path),os.Getpid(),params)
}
