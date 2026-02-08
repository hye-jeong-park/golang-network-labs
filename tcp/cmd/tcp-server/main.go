package main

import (
	"log"
	"os"

	"golang-network-labs/tcp/internal/server"
)

func main() {
	// 리슨 포트 기본값
	port := os.Getenv("TCP_PORT")
	// 없으면 9000
	if port == "" {
		port = "9000"
	}

	// 서버 생성
	s := server.New(server.Config{
		Addr: ":" + port,
	})

	// 시작 로그
	log.Println("tcp :", port)

	// 서버 실행
	if err := s.ListenAndServe(); err != nil {
		// 치명 에러면 종료
		log.Fatal(err)
	}
}
