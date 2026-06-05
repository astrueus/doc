package commands

import (
	"fmt"
	"os"

	"git.itopcms.com/jackliu/doc/conf"
)

// CheckUpdate 显示当前版本（自定义维护，不检查上游更新）.
func CheckUpdate() {
	fmt.Println("Doc current version => ", conf.VERSION)
	os.Exit(0)
}
