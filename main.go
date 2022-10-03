package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron/v3"
)

type cronResult struct {
	ID     int
	Time   string
	types  string
	typeID string
}

func main() {
	fmt.Println("============= Conectado al server 4000 ==========")

	watcher, errWatcher := fsnotify.NewWatcher()
	if errWatcher != nil {
		log.Fatal(errWatcher)
	}

	defer watcher.Close()

	done := make(chan bool)
	go watchFiles(watcher, done)

	errCrontab := watcher.Add("./crontab")

	if errCrontab != nil {
		log.Fatal("Add failed:", errCrontab)
	}

	<-done

	err := http.ListenAndServe(":4000", nil)

	if err != nil {
		log.Fatalf("Ocurrió un error con el server.")
		log.Fatal(err)
	}
}

func watchFiles(watcher *fsnotify.Watcher, done chan bool) {
	defer close(done)

	runCronTab()

	var (
		timer     *time.Timer
		lastEvent fsnotify.Event
	)

	timer = time.NewTimer(time.Millisecond)
	<-timer.C // timer should be expired at first

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// log.Printf("El archivo ha cambiado %s %s\n", event.Name, event.Op)
			lastEvent = event
			timer.Reset(time.Millisecond * 100)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)

		case <-timer.C:
			if lastEvent.Op&fsnotify.Write == fsnotify.Write {
				runCronTab()
			}

		}
	}
}

func runCronTab() {
	time.Sleep(2000)

	fileInfo, _ := os.Stat("./crontab")

	if fileInfo.Size() == 0 {
		writeLog(false, "El archivo está vacío.")
		fmt.Println("El archivo está vacío.")
		return
	}

	data, err := os.ReadFile("./crontab")

	if err != nil {
		writeLog(false, "Ha ocurrido un error al leer el archivo.")
		log.Fatalf("Ha ocurrido un error al leer el archivo.")
		log.Fatal(err)
	}

	array := strings.Split(string(data), "\n")

	var info []cronResult = []cronResult{}

	for i := 0; i < len(array); i++ {
		values := strings.Split(array[i], "_")

		var jsonStr = cronResult{
			ID:     i,
			Time:   values[0],
			types:  values[3],
			typeID: values[4],
		}

		info = append(info, jsonStr)

		c := cron.New()

		fmt.Println(info)

		c.AddFunc(values[0], func() {
			fmt.Printf("Run cron in %v", time.Now())

			fmt.Println("")

			if values[3] == "notification:send" {
				fmt.Println("Enviando notificación de ID: " + values[4])
				writeLog(true, "Notificación enviada con ID: "+values[4])
			}

			if values[3] == "cart-notification:send" {
				fmt.Println("Enviando notificación de carrito olvidado ID: " + values[4])
				writeLog(true, "Notificación de carrito olvidado enviada con ID: "+values[4])
			}
		})

		c.Start()
	}
}

func writeLog(success bool, text string) {
	file, err := os.OpenFile("./logs/notifications-log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err != nil {
		log.Fatalln(err)
	}

	log.SetOutput(file)
	log.Printf("SUCCESS: %v, MESSAGE: %s", success, text)

}
