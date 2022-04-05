package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"health"
)

func main() {
	srv := fasthttp.Server{
		Handler: Handler,
		// srv.Shutdown does not drop keepalive connects
		// unless Read/Idle timeouts are provided.
		ReadTimeout: time.Second,
		IdleTimeout: time.Second,
	}

	health.SetDownDelay(time.Second * 5)
	health.AddDownFn(func() {
		fmt.Println("sigint")
	})
	health.AddDownFn(func() {
		if err := srv.Shutdown(); err != nil {
			log.Println(err)
		}
	})

	// Immediately returns nil on srv.Shutdown.
	if err := srv.ListenAndServe(":8080"); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	health.WaitDown(ctx)
	cancel()
}

var cnt = uint32(0)

func Handler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	switch string(ctx.Path()) {
	case "/health":
		_, _ = ctx.WriteString(fmt.Sprintf(`{"health":%v}`, health.Check()))

	case "/stop":
		health.Stop()
		_, _ = ctx.WriteString(`{"ok":true}`)

	default:
		i := atomic.AddUint32(&cnt, 1)
		time.Sleep(time.Second * time.Duration(i))
		_, _ = ctx.WriteString(fmt.Sprintf(`{"i":%v}`, i))
	}
}
