package main

import (
	"fmt"
	"os"
	_ "time/tzdata"

	"git.itopcms.com/jackliu/doc/commands"
	"git.itopcms.com/jackliu/doc/commands/daemon"
	_ "git.itopcms.com/jackliu/doc/routers"
	_ "github.com/beego/beego/v2/server/web/session/memcache"
	_ "github.com/beego/beego/v2/server/web/session/mysql"
	_ "github.com/beego/beego/v2/server/web/session/redis"
	"github.com/kardianos/service"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	commands.StartWebServer = func(flagArgs []string) error {
		d := daemon.NewDaemon()
		d.Config().Arguments = flagArgs
		s, err := service.New(d, d.Config())
		if err != nil {
			return fmt.Errorf("create service: %w", err)
		}
		if err := s.Run(); err != nil {
			return fmt.Errorf("启动程序失败 -> %w", err)
		}
		return nil
	}
}

func main() {
	// System service management bypasses cobra to keep old scripts working:
	//   doc service install|remove|restart
	if len(os.Args) >= 3 && os.Args[1] == "service" {
		switch os.Args[2] {
		case "install":
			daemon.Install()
		case "remove":
			daemon.Uninstall()
		case "restart":
			daemon.Restart()
		default:
			fmt.Fprintf(os.Stderr, "unknown service command: %s\n", os.Args[2])
			os.Exit(1)
		}
	}

	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
