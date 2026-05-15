// cmd/genpw 生成 bcrypt 密码哈希，用于为 ROUNDNFC_ADMIN_PASSWORD_HASH 填值。
//
// 用法：
//
//	go run ./cmd/genpw           # 交互式输入
//	go run ./cmd/genpw "my pw"  # 参数传入。注意 shell 历史会看到
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"backend-go/internal/auth"
)

func main() {
	var pw string
	if len(os.Args) > 1 {
		pw = os.Args[1]
	} else {
		fmt.Print("password: ")
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		pw = strings.TrimRight(s, "\r\n")
	}
	if pw == "" {
		fmt.Fprintln(os.Stderr, "empty password")
		os.Exit(1)
	}
	h, err := auth.HashPassword(pw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(h)
}
