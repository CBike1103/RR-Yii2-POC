package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/spiral/roadrunner"
	rrhttp "github.com/spiral/roadrunner/service/http"
	rrstatic "github.com/spiral/roadrunner/service/static"
)

func main() {
	serveHTTP()
}

func plainTest() {
	HelloServer := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "hello, world!\n")
	}

	http.HandleFunc("/", HelloServer)
	err := http.ListenAndServe(":8013", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveHTTP() {
	httpServer := &rrhttp.Service{}
	serverCfg := &rrhttp.Config{
		Address: ":8013",
		Workers: &roadrunner.ServerConfig{
			// Command: "php -d zend_extension=xdebug.so -d xdebug.remote_enable=1 -d xdebug.remote_autostart=On -d xdebug.idekey=VSCODE /home/chenfang/Codes/php/trial/psr7/psrtest/roadrunner.php",
			Command: "php /home/chenfang/Codes/php/trial/psr7/psrtest/roadrunner.php",
			// Command: "php /home/chenfang/Codes/php/support-csm.gtarcade.com/roadrunner.php",
			// Command: "php psr.php",
			// Command: "php7.1 /home/chenfang/Codes/php//roadrunner.php",
			Relay: "pipe",
			Pool: &roadrunner.Config{
				NumWorkers:      4,
				MaxJobs:         0,
				AllocateTimeout: 60 * time.Second,
				DestroyTimeout:  60 * time.Second,
			},
		},
	}
	serverCfg.Workers.SetEnv("YII_ALIAS_WEBROOT", "/home/chenfang/Codes/php/trial/psr7/psrtest/web")
	serverCfg.Workers.SetEnv("YII_ALIAS_WEB", "http://localhost:8013")
	_, err := httpServer.Init(serverCfg, nil, nil)
	if err != nil {
		panic(err)
	}

	staticServer := &rrstatic.Service{}
	staticCfg := &rrstatic.Config{
		Dir: "/home/chenfang/Codes/php/trial/psr7/psrtest/web",
	}
	staticServer.Init(staticCfg, httpServer)

	err = httpServer.Serve()
	if err != nil {
		panic(err)
	}
}
