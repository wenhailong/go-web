package main

import (
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"os"
	"time"
)

func generateStatusData(fans int64, progress int64) bool {
	return fans == progress
}

func main() {
	session, err := mgo.Dial("mongodb://192.168.158.70:27000")
	if err != nil {
		fmt.Println("mgo.Dial failed. error=%v", err)
		os.Exit(1)
	}

	collection := session.DB("follower").C("user")
	bulk := collection.Bulk()

	for i := 0; i < 100; i++ {
		doc := make(bson.M)
		doc["userId"] = generateUserId(i)
		doc["coins"] = int64(rand.Int() % 200)
		doc["lastPushDate"] = int64(0)

		var orders []bson.M
		for j := 0; j < rand.Int()%4; j++ {
			order := make(bson.M)
			order["orderId"] = generateOrderIdData()
			order["date"] = generateOrderTimeData()
			order["coins"] = generateCoinData()
			order["fans"] = generateFansData(order["coins"].(int64))
			order["progress"] = generateProgressData(order["fans"].(int64))
			order["status"] = generateStatusData(order["fans"].(int64), order["progress"].(int64))
			orders = append(orders, order)
		}

		doc["orders"] = orders

		bulk.Insert(doc)
	}

	_, err = bulk.Run()
	if err != nil {
		fmt.Println("bulk.Run failed. error=%v", err)
	} else {
		fmt.Println("bulk.Run finish")
	}

}

func generateProgressData(fans int64) int64 {
	return fans - rand.Int63()%fans
}

func generateFansData(coin int64) int64 {
	return coin * 100
}

func generateCoinData() int64 {
	return rand.Int63()%10 + 1
}

func generateOrderIdData() string {
	return fmt.Sprintf("%v", uuid.NewV4())
}

func generateOrderTimeData() int64 {
	now := time.Now()
	orderTime := now.AddDate(0, 0, rand.Int()%30)
	return orderTime.Unix()
}

func generateUserId(base int) string {
	return fmt.Sprintf("%09d", base)
}
