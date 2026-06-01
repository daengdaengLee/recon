package main

import (
	"os"
	"recon/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		// cobra가 Execute() 내부에서 이미 에러를 stderr에 출력하므로
		// 여기서는 비정상 종료 코드만 반환한다.
		os.Exit(1)
	}
}
