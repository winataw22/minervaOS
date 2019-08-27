package main

import (
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"io"
	"net"
	"strings"
)

func reader(c io.Reader, r *redis.Client, channel string) {
	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf[:])
		if err != nil {
			return
		}

		logline := strings.TrimSpace(string(buf[0:n]))

		// println(">> ", logline)
		if r.Publish(channel, logline).Err() != nil {
			panic(err)
		}
	}
}

func main() {
	lUnix := flag.String("logs", "/var/run/log.sock", "zinit unix socket")
	rChan := flag.String("channel", "zinit-logs", "redis logs channel name")
	rHost := flag.String("host", "localhost", "redis host")
	rPort := flag.Int("port", 6379, "redis port")

	flag.Parse()

	fmt.Printf("[+] opening logs: %s\n", *lUnix)

	// connect to local logs
	c, err := net.Dial("unix", *lUnix)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("[+] connecting redis: %s:%d\n", *rHost, *rPort)

	// connect to redis server
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", *rHost, *rPort),
		Password: "",
		DB:       0,
	})

	if _, err := client.Ping().Result(); err != nil {
		panic(err.Error())
	}

	fmt.Printf("[+] forwarding logs to channel: %s\n", *rChan)

	reader(c, client, *rChan)
}
