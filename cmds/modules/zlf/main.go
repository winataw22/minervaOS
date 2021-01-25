package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
)

func reader(c io.Reader, r *redis.Client, channel string) {
	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf[:])
		if err != nil {
			return
		}

		logline := strings.TrimSpace(string(buf[0:n]))

		if r.Publish(channel, logline).Err() != nil {
			log.Fatal().Err(err).Msg("error while publishing log line")
		}
	}
}

func main() {
	app.Initialize()

	lUnix := flag.String("logs", "/var/run/log.sock", "zinit unix socket")
	rChan := flag.String("channel", "zinit-logs", "redis logs channel name")
	rHost := flag.String("host", "localhost", "redis host")
	rPort := flag.Int("port", 6379, "redis port")

	flag.Parse()

	fmt.Printf("[+] opening logs: %s\n", *lUnix)

	// connect to local logs
	c, err := net.Dial("unix", *lUnix)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot dial server")
	}

	fmt.Printf("[+] connecting redis: %s:%d\n", *rHost, *rPort)

	// connect to redis server
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", *rHost, *rPort),
		Password: "",
		DB:       0,
	})

	if _, err := client.Ping().Result(); err != nil {
		log.Fatal().Err(err).Msg("cannot ping server")
	}

	fmt.Printf("[+] forwarding logs to channel: %s\n", *rChan)

	reader(c, client, *rChan)
}
