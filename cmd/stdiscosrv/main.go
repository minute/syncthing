// Copyright (C) 2014-2015 Jakob Borg and Contributors (see the CONTRIBUTORS file).

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/thejerf/suture"
)

var (
	Version    string
	BuildStamp string
	BuildUser  string
	BuildHost  string

	BuildDate   time.Time
	LongVersion string
)

func init() {
	stamp, _ := strconv.Atoi(BuildStamp)
	BuildDate = time.Unix(int64(stamp), 0)

	date := BuildDate.UTC().Format("2006-01-02 15:04:05 MST")
	LongVersion = fmt.Sprintf(`stdiscosrv %s (%s %s-%s) %s@%s %s`, Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, BuildUser, BuildHost, date)
}

var (
	lruSize     = 10240
	limitAvg    = 5
	limitBurst  = 20
	globalStats stats
)

func main() {
	const (
		cleanIntv = 1 * time.Hour
		statsIntv = 5 * time.Minute
	)

	var listen string

	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	flag.StringVar(&listen, "listen", "127.0.0.1:8888", "Listen address")
	flag.IntVar(&lruSize, "limit-cache", lruSize, "Limiter cache entries")
	flag.IntVar(&limitAvg, "limit-avg", limitAvg, "Allowed average request rate, per 10 s")
	flag.IntVar(&limitBurst, "limit-burst", limitBurst, "Allowed burst size, requests")
	flag.Parse()

	log.Println(LongVersion)

	main := suture.NewSimple("main")

	main.Add(&querysrv{
		addr: listen,
	})

	main.Add(&statssrv{
		intv: statsIntv,
	})

	globalStats.Reset()
	main.Serve()
}

func next(intv time.Duration) time.Duration {
	t0 := time.Now()
	t1 := t0.Add(intv).Truncate(intv)
	return t1.Sub(t0)
}
