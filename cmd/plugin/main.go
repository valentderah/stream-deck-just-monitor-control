package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/valentderah/stream-deck-just-monitor-control/internal/actions"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/display"
	"github.com/valentderah/stream-deck-just-monitor-control/internal/streamdeck"
)

func main() {
	port := flag.Int("port", 0, "Stream Deck WebSocket port")
	pluginUUID := flag.String("pluginUUID", "", "Plugin UUID")
	registerEvent := flag.String("registerEvent", "", "Register event name")
	info := flag.String("info", "", "Stream Deck info JSON")
	listMonitors := flag.Bool("list-monitors", false, "list monitors and exit")
	debug := flag.Bool("debug", false, "verbose stderr logging")
	flag.Parse()

	_ = info
	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(os.Stderr)
	}

	mgr := display.NewManager()

	if *listMonitors {
		if err := runListMonitors(mgr); err != nil {
			fmt.Fprintf(os.Stderr, "list-monitors: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *port <= 0 {
		fmt.Fprintf(os.Stderr, "usage: plugin.exe -port <n> -pluginUUID <id> -registerEvent <event> [-info <json>]\n")
		fmt.Fprintf(os.Stderr, "   or: plugin.exe --list-monitors [--debug]\n")
		os.Exit(2)
	}

	client, err := streamdeck.Connect(*port, *pluginUUID, *registerEvent)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Close()

	r := streamdeck.NewRouter()
	r.Register("com.valentderah.just-monitor-control.brightness", actions.NewBrightness(mgr, client))
	r.Register("com.valentderah.just-monitor-control.input-switch", actions.NewInputSwitch(mgr, client))
	r.Register("com.valentderah.just-monitor-control.refresh-rate", actions.NewRefreshRate(mgr, client))
	r.Register("com.valentderah.just-monitor-control.hdr", actions.NewHDR(mgr, client))
	r.Register("com.valentderah.just-monitor-control.sleep", actions.NewSleep(mgr, client))
	r.Register("com.valentderah.just-monitor-control.raw-vcp", actions.NewRawVCP(mgr, client))

	if *debug {
		log.Printf("plugin registered uuid=%s", *pluginUUID)
	}

	for {
		ev, err := client.ReadEvent()
		if err != nil {
			if *debug {
				log.Printf("read exit: %v", err)
			}
			return
		}
		if *debug {
			b, _ := json.Marshal(ev)
			log.Printf("event: %s", b)
		}
		r.Handle(ev)
	}
}

func runListMonitors(mgr display.Manager) error {
	mons, err := mgr.GetMonitors(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("id\tname\tbrightness\tport")
	for _, m := range mons {
		fmt.Printf("%s\t%s\t%d\t%d\n", m.ID, m.Name, m.Brightness, m.CurrentPort)
	}
	return nil
}
