package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var g_mgoSession *mgo.Session

func main() {
	initialize()
	startHttp()
}

func startHttp() {
	http.HandleFunc("/getfollowers/coins", errorHandler(coinsHandler))
	http.HandleFunc("/getfollowers/info", errorHandler(infoHandler))
	http.HandleFunc("/getfollowers/buyfollower", errorHandler(buyfollowerHandler))
	http.HandleFunc("/getfollowers/getuser", errorHandler(getuserHandler))
	http.HandleFunc("/getfollowers/progress", errorHandler(progressHandler))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func initialize() {
	initMongo()
	initLog()
	loadBuyerData()
}

func initMongo() {
	mgoUri := "mongodb://192.168.158.70:27000"
	session, err := mgo.Dial(mgoUri)
	if err != nil {
		fmt.Println("[initMongo] connect to mongodb failed. mongoUri=", mgoUri)
		os.Exit(1)
	}

	g_mgoSession = session
}

func initLog() {
	logPath := "log"
	if _, err := os.Stat(logPath); err != nil {
		err = os.Mkdir(logPath, os.ModeDir)
		if err != nil {
			fmt.Println("create log forlder failed. error=%v", err)
			os.Exit(1)
		}
	}

	now := time.Now()
	logSuffix := fmt.Sprintf("-%d-%02d-%02dT%02d%02d%02d.log", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	logName := filepath.Base(os.Args[0])
	logName = logPath + "\\" + logName + logSuffix

	f, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
}

func loadBuyerData() {
	collection := g_mgoSession.DB("follower").C("user")
	queryStatement := bson.M{"orders": bson.M{"$elemMatch": bson.M{"status": false}}}
	iter := collection.Find(queryStatement).Iter()

	var result Buyer
	var counter int
	for iter.Next(&result) {
		g_buyerManager[result.UserId] = result
		counter++
	}

	if err := iter.Close(); err != nil {
		fmt.Printf("load buyerdata failed. err=%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("load buyerdata success. count:%d\n", counter)
}

func errorHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err, ok := recover().(error); ok {
				log.Error(err.Error())
				responseError(w, err)
			}
		}()

		fn(w, r)
	}
}

func responseError(w http.ResponseWriter, err error) {
	var response = struct {
		Err string
	}{
		err.Error(),
	}

	respByte, err := json.Marshal(response)
	if err != nil {
		log.Error("[responseError] json.Marshaler failed. error=%v", err.Error())
		return
	}

	_, err = io.WriteString(w, string(respByte))
	if err != nil {
		log.Error("[responseError] io.WriteString failed. error=%v", err.Error())
		return

	}
}
