package main

import (
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"time"
)

func main() {
	session, err := mgo.Dial("mongodb://192.168.158.70:27000")
	if err != nil {

	}

	collection := session.DB("follower").C("user")
	bulk := collection.Bulk()

	for i := 0; i < 100; i++ {
		doc := make(bson.M)
		doc["userid"] = generateUserId(i)
		doc["clickCoins"] = rand.Int() % 200

		var orders []bson.M
		for j := 0; j < rand.Int()%4; j++ {
			order := make(bson.M)
			order["orderid"] = generateUseridData()
			order["date"] = generateOrderTimeData()
			order["coins"] = generateCoinData()
			order["fans"] = generateFansData(order["coins"].(int))
			order["progress"] = generateProgressData(order["fans"].(int))
			order["status"] = generateStatusData(order["fans"].(int), order["progress"].(int))
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

func generateStatusData(fans int, progress int) bool {
	return fans == progress
}

func generateProgressData(fans int) int {
	return fans - rand.Int()%fans
}

func generateFansData(coin int) int {
	return coin * 100
}

func generateCoinData() int {
	return rand.Int()%10 + 1
}

func generateUseridData() string {
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
