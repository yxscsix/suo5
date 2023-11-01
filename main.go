package main

import (
	"context"
	"fmt"
	log "github.com/kataras/golog"
	"github.com/urfave/cli/v2"
	"github.com/zema1/suo5/ctrl"
	"os"
	"os/signal"
	"strings"
)

var Version = "v0.0.0"

func main() {
	log.Default.SetTimeFormat("01-02 15:04")
	app := cli.NewApp()
	app.Name = "suo5"
	app.Usage = "A super http proxy tunnel"
	app.Version = Version

	defaultConfig := ctrl.DefaultSuo5Config()

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "target",
			Aliases:  []string{"t"},
			Usage:    "set the remote server url, ex: http://localhost:8080/tomcat_debug_war_exploded/",
			Value:    defaultConfig.Target,
			Required: true,
		},
		&cli.StringFlag{
			Name:    "listen",
			Aliases: []string{"l"},
			Usage:   "set the listen address of socks5 server",
			Value:   defaultConfig.Listen,
		},
		&cli.StringFlag{
			Name:    "method",
			Aliases: []string{"m"},
			Usage:   "http request method",
			Value:   defaultConfig.Method,
		},
		&cli.StringFlag{
			Name:    "redirect",
			Aliases: []string{"r"},
			Usage:   "redirect to the url if host not matched, used to bypass load balance",
			Value:   defaultConfig.RedirectURL,
		},
		&cli.BoolFlag{
			Name:  "no-auth",
			Usage: "disable socks5 authentication",
			Value: defaultConfig.NoAuth,
		},
		&cli.StringFlag{
			Name:  "auth",
			Usage: "socks5 creds, username:password, leave empty to auto generate",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mode",
			Usage: "connection mode, choices are auto, full, half",
			Value: string(defaultConfig.Mode),
		},
		&cli.StringFlag{
			Name:  "ua",
			Usage: "the user-agent used to send request",
			Value: `Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.1.2.3\rProxy: UP`,
		},
		&cli.StringSliceFlag{
			Name:    "header",
			Aliases: []string{"H"},
			Usage:   "use extra header, ex -H 'Cookie: abc'",
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "http request timeout in seconds",
			Value: defaultConfig.Timeout,
		},
		&cli.IntFlag{
			Name:  "buf-size",
			Usage: "set the request max body size",
			Value: defaultConfig.BufferSize,
		},
		&cli.StringFlag{
			Name:  "proxy",
			Usage: "use upstream proxy, support both socks5 and http(s), eg: socks5://127.0.0.1:7890",
			Value: defaultConfig.UpstreamProxy,
		},
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "debug the traffic, print more details",
			Value:   defaultConfig.Debug,
		},
		&cli.BoolFlag{
			Name:    "no-heartbeat",
			Aliases: []string{"nh"},
			Usage:   "disable heartbeat to the remote server which will send data every 5s",
			Value:   defaultConfig.DisableHeartbeat,
		},
		&cli.BoolFlag{
			Name:    "no-gzip",
			Aliases: []string{"ng"},
			Usage:   "disable gzip compression, which will improve compatibility with some old servers",
			Value:   defaultConfig.DisableHeartbeat,
		},
		&cli.StringFlag{
			Name:    "test-exit",
			Aliases: []string{"T"},
			Usage:   "test a real connection, if success exit(0), else exit(1)",
			Hidden:  true,
		},
	}
	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			log.Default.SetLevel("debug")
		}
		return nil
	}
	app.Action = Action

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func Action(c *cli.Context) error {
	listen := c.String("listen")
	target := c.String("target")
	noAuth := c.Bool("no-auth")
	auth := c.String("auth")
	mode := ctrl.ConnectionType(c.String("mode"))
	ua := c.String("ua")
	bufSize := c.Int("buf-size")
	timeout := c.Int("timeout")
	debug := c.Bool("debug")
	proxy := c.String("proxy")
	method := c.String("method")
	redirect := c.String("redirect")
	header := c.StringSlice("header")
	noHeartbeat := c.Bool("no-heartbeat")
	noGzip := c.Bool("no-gzip")
	testExit := c.String("test-exit")

	var username, password string
	if auth == "" {
		username = "suo5"
		password = ctrl.RandString(8)
	} else {
		parts := strings.Split(auth, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid socks credentials, expected username:password")
		}
		username = parts[0]
		password = parts[1]
		noAuth = false
	}
	if !(mode == ctrl.AutoDuplex || mode == ctrl.FullDuplex || mode == ctrl.HalfDuplex) {
		return fmt.Errorf("invalid mode, expected auto or full or half")
	}

	if bufSize < 512 || bufSize > 1024000 {
		return fmt.Errorf("inproper buffer size, 512~1024000")
	}
	header = append(header, "User-Agent: "+ua)

	config := &ctrl.Suo5Config{
		Listen:           listen,
		Target:           target,
		NoAuth:           noAuth,
		Username:         username,
		Password:         password,
		Mode:             mode,
		BufferSize:       bufSize,
		Timeout:          timeout,
		Debug:            debug,
		UpstreamProxy:    proxy,
		Method:           method,
		RedirectURL:      redirect,
		RawHeader:        header,
		DisableHeartbeat: noHeartbeat,
		DisableGzip:      noGzip,
		TestExit:         testExit,
	}
	ctx, cancel := signalCtx()
	defer cancel()
	return ctrl.Run(ctx, config)
}

func signalCtx() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}
