package main

import (
	"encoding/json"
	"flag"
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

var gobalMgoSession *mgo.Session

func main() {
	parseCommandLine()

	initLog()
	initMongo()
	loadUserOrders()
	startCounter()

	startHttp()
}

func parseCommandLine() {
	requiredArgs := 2

	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS] \noptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&gobalConfig.IP, "bind_ip", "", "http server ip. (Required)")
	flag.IntVar(&gobalConfig.Port, "port", DefaultPort, "http server port.")
	flag.StringVar(&gobalConfig.MongoUri, "mongoUri", "", "mongodb uri. (Required)")
	flag.Parse()

	if flag.NFlag() != requiredArgs {
		flag.Usage()
	}
}

func startHttp() {
	http.HandleFunc("/counter", counterHander)

	http.HandleFunc("/getfollowers/coins", Decorate(coinsHandler, loggingAndRespError(), counting(&gobalCounter)))
	http.HandleFunc("/getfollowers/info", Decorate(infoHandler, loggingAndRespError(), counting(&gobalCounter)))
	http.HandleFunc("/getfollowers/buyfollower", Decorate(buyfollowerHandler, loggingAndRespError(), counting(&gobalCounter)))
	http.HandleFunc("/getfollowers/getuser", Decorate(getUserHandler, loggingAndRespError(), counting(&gobalCounter)))
	http.HandleFunc("/getfollowers/progress", Decorate(progressHandler, loggingAndRespError(), counting(&gobalCounter)))

	log.Infof("start http server. ip:%v port=%v", gobalConfig.IP, gobalConfig.Port)

	addr := fmt.Sprintf("%v:%v", gobalConfig.IP, gobalConfig.Port)
	err := http.ListenAndServeTLS(addr, "server.crt", "server.key", nil)
	if err != nil {
		log.Errorf(fmt.Sprintf("[startHttp] http.ListenAndServe failed. error=%v", err))
		os.Exit(1)
	}
}

func initMongo() {
	session, err := mgo.Dial(gobalConfig.MongoUri)
	if err != nil {
		log.Errorf("[initMongo] connect to mongodb failed. mongoUri=%v", gobalConfig.MongoUri)
		os.Exit(1)
	}

	gobalMgoSession = session
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
	logName = logPath + "//" + logName + logSuffix

	f, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.InfoLevel)
}

func loadUserOrders() {
	collection := gobalMgoSession.DB("follower").C("user")
	queryStatement := bson.M{"orders": bson.M{"$elemMatch": bson.M{"status": false}}}
	iter := collection.Find(queryStatement).Select(bson.M{"_id": 0, "userId": 1, "orders": 1}).Iter()

	var result bson.M
	var counter int
	for iter.Next(&result) {
		if orders, ok := result["orders"]; ok {
			for _, order := range orders.([]interface{}) {

				item := PushItem{&Order{
					order.(bson.M)["orderId"].(string),
					order.(bson.M)["coins"].(int64),
					order.(bson.M)["date"].(int64),
					order.(bson.M)["progress"].(int64),
					order.(bson.M)["fans"].(int64),
					order.(bson.M)["status"].(bool),
				}, result["userId"].(string)}

				gobalPushManger.Add(&item)
				counter++
			}
		}
	}

	if err := iter.Close(); err != nil {
		log.Errorf("load user orders failed. err=%v", err)
		os.Exit(1)
	}

	log.Infof("load user orders success. count:%d", counter)
}

func startCounter() {
	go func(c *Counter) {
		lastRequest := c.Request()

		for {
			time.Sleep(1 * time.Second)

			curRequest := c.Request()
			c.SetRequestPerSecond(curRequest - lastRequest)
			lastRequest = curRequest
		}
	}(&gobalCounter)
}

func responseError(w http.ResponseWriter, err error) {
	err = err.(FollowerError)
	respByte, err := json.Marshal(err)
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
