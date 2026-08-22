package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"taskt111-shortlink/internal/httpapi"
	"taskt111-shortlink/internal/store"
)

func main() {
	var (
		addr   = flag.String("addr", ":8080", "HTTP 监听地址")
		dbPath = flag.String("db", "shortlink.db", "SQLite 数据库文件路径")
		smoke  = flag.Bool("smoke-test", false, "运行内置自检并退出")
	)
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(); err != nil {
			fmt.Fprintln(os.Stderr, "smoke-test FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("smoke-test OK")
		os.Exit(0)
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	h := httpapi.NewHandler(st)
	srv := &http.Server{Addr: *addr, Handler: h.Routes()}
	log.Printf("shortlink listening on %s (db=%s)", *addr, *dbPath)
	log.Fatal(srv.ListenAndServe())
}
