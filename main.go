package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/beego/beego/v2/server/web/session/memcache"
	_ "github.com/beego/beego/v2/server/web/session/mysql"
	_ "github.com/beego/beego/v2/server/web/session/redis"
	"github.com/kardianos/service"
	_ "github.com/mattn/go-sqlite3"
	"git.itopcms.com/jackliu/doc/commands"
	"git.itopcms.com/jackliu/doc/commands/daemon"
	_ "git.itopcms.com/jackliu/doc/routers"
)

func main() {

	if len(os.Args) >= 3 && os.Args[1] == "service" {
		if os.Args[2] == "install" {
			daemon.Install()
		} else if os.Args[2] == "remove" {
			daemon.Uninstall()
		} else if os.Args[2] == "restart" {
			daemon.Restart()
		}
	}
	commands.RegisterCommand()

	d := daemon.NewDaemon()

	s, err := service.New(d, d.Config())

	if err != nil {
		fmt.Println("Create service error => ", err)
		os.Exit(1)
	}

	if err := s.Run(); err != nil {
		log.Fatal("启动程序失败 ->", err)
	}
}
