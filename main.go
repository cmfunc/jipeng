package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cmfunc/jipeng/cache"
	"github.com/cmfunc/jipeng/conf"
	"github.com/cmfunc/jipeng/db"
	"github.com/cmfunc/jipeng/mq"
	"github.com/cmfunc/jipeng/router"
	"github.com/cmfunc/jipeng/wx"
)

func main() {
	conf.ParseJipengConf()
	cache.Init(conf.Get().Redis)
	db.InitMySQL(conf.Get().MySQL)
	db.InitMongo(conf.Get().MongoDB)
	mq.Init(conf.Get().NSQ)
	wx.Set(conf.Get().Wx)

	engine := gin.Default()
	router.Inject(engine)
	srv := &http.Server{
		Addr:              ":7777",
		Handler:           engine,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       10 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          &log.Logger{},
	}
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("Server exiting")
}
