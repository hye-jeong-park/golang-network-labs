package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// 전체 설정 묶음
type Config struct {
	DB   DBConfig
	TCP  TCPConfig
	HTTP HTTPConfig
	Run  RunConfig
	Rate RateConfig
}

// DB 설정
type DBConfig struct {
	Host string
	Port string
	Name string
	User string
	Pass string
}

// TCP 설정
type TCPConfig struct {
	Host        string
	Port        string
	DialTimeout time.Duration
	IOTimeout   time.Duration
}

// HTTP 설정
type HTTPConfig struct {
	Timeout time.Duration
}

// /run 동시 실행 제한
type RunConfig struct {
	MaxConcurrency int
}

// IP RateLimit 설정
type RateConfig struct {
	RPS   float64
	Burst int
}

// 초 단위 환경변수 → Duration
func envSeconds(key string, defSec int) time.Duration {
	// 공백 제거
	v := strings.TrimSpace(os.Getenv(key))
	// 없으면 기본값
	if v == "" {
		return time.Duration(defSec) * time.Second
	}
	// 정수 파싱
	n, err := strconv.Atoi(v)
	// 실패/0이하면 기본값
	if err != nil || n <= 0 {
		return time.Duration(defSec) * time.Second
	}
	// 초 단위 반환
	return time.Duration(n) * time.Second
}

// 정수 환경변수
func envInt(key string, def int) int {
	// 공백 제거
	v := strings.TrimSpace(os.Getenv(key))
	// 없으면 기본값
	if v == "" {
		return def
	}
	// 정수 파싱
	n, err := strconv.Atoi(v)
	// 실패/0이하면 기본값
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// 실수 환경변수
func envFloat(key string, def float64) float64 {
	// 공백 제거
	v := strings.TrimSpace(os.Getenv(key))
	// 없으면 기본값
	if v == "" {
		return def
	}
	// 실수 파싱
	f, err := strconv.ParseFloat(v, 64)
	// 실패/0이하면 기본값
	if err != nil || f <= 0 {
		return def
	}
	return f
}

// 환경변수 로드
func Load() Config {
	dbHost := strings.TrimSpace(os.Getenv("DB_HOST"))
	dbPort := strings.TrimSpace(os.Getenv("DB_PORT"))
	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))
	dbUser := strings.TrimSpace(os.Getenv("DB_USER"))
	dbPass := strings.TrimSpace(os.Getenv("DB_PASS"))

	if dbHost == "" || dbPort == "" || dbName == "" || dbUser == "" || dbPass == "" {
		panic("DB 환경변수가 누락되었습니다(DB_HOST/DB_PORT/DB_NAME/DB_USER/DB_PASS)")
	}

	// TCP
	tcpHost := strings.TrimSpace(os.Getenv("TCP_HOST"))
	tcpPort := strings.TrimSpace(os.Getenv("TCP_PORT"))
	if tcpHost == "" {
		tcpHost = "tcp"
	}
	if tcpPort == "" {
		tcpPort = "9000"
	}

	// Timeout
	httpTimeout := envSeconds("HTTP_TIMEOUT_SEC", 5)
	dialTimeout := envSeconds("TCP_DIAL_TIMEOUT_SEC", 2)
	ioTimeout := envSeconds("TCP_IO_TIMEOUT_SEC", 5)

	// /run 동시 실행 제한 (기본 5)
	maxConc := envInt("RUN_MAX_CONCURRENCY", 5)

	// IP 레이트리밋 기본값
	rps := envFloat("RATE_RPS", 5)
	burst := envInt("RATE_BURST", 10)

	// 설정 묶어서 반환
	return Config{
		DB: DBConfig{
			Host: dbHost,
			Port: dbPort,
			Name: dbName,
			User: dbUser,
			Pass: dbPass,
		},
		TCP: TCPConfig{
			Host:        tcpHost,
			Port:        tcpPort,
			DialTimeout: dialTimeout,
			IOTimeout:   ioTimeout,
		},
		HTTP: HTTPConfig{
			Timeout: httpTimeout,
		},
		Run: RunConfig{
			MaxConcurrency: maxConc,
		},
		Rate: RateConfig{
			RPS:   rps,
			Burst: burst,
		},
	}
}
