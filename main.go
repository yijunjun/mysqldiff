// mysqldiff project main.go
package main

import (
	"flag"
	"fmt"
)

// 编译时修改
var (
	// githash值
	GITHASH = "unknown"

	// 编译时间
	COMPILETIME = "unknown"

	G_Left_Json string

	G_Right_Json string

	G_Port string
)

func main() {
	flag.StringVar(&G_Left_Json, "leftJson", "left.json", "left json conf")
	flag.StringVar(&G_Right_Json, "rightJson", "right.json", "right json conf")
	flag.StringVar(&G_Port, "port", ":8086", "listen port")

	flag.Parse()

	err := LoadMysql(G_Left_Json, G_Right_Json)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(Web(G_Port))
}
