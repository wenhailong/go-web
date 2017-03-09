package main

import (
	"encoding/json"
	"fmt"
	"github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"net/http/httptest"
	"testing"
	"time"
)

var testGobalHttpAddr = "192.168.158.70:8080"
var testGobalMongoUri = "mongodb://192.168.158.70:27000"
var testGobalDbName = "follower"
var testGobalCollName = "user"
var testGobalVersion = 1
var testGobalUserIdNotExist = "999999999"

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&FollowerHandlerSuite{})

type FollowerHandlerSuite struct {
	userId     string
	Coins      int64
	OrderCount int
	OrderIds   []string
	session    *mgo.Session
}

func (p *FollowerHandlerSuite) SetUpTest(c *C) {
	orderCount := 2
	coins := int64(20)
	userId := fmt.Sprintf("%09d", 200)
	orderIds := []string{
		fmt.Sprintf("%v", uuid.NewV4()),
		fmt.Sprintf("%v", uuid.NewV4()),
	}

	session, err := mgo.Dial(testGobalMongoUri)
	if err != nil {
		c.Fatal(err)
	}
	collection := session.DB(testGobalDbName).C(testGobalCollName)
	p.cleanTestDataIfExist(c, collection, userId)

	doc := p.CreateTestDoc(userId, coins, orderIds)
	err = collection.Insert(doc)
	if err != nil {
		c.Fatal(err)
	}

	p.OrderIds = orderIds
	p.OrderCount = orderCount
	p.Coins = coins
	p.userId = userId
	p.session = session.Copy()

	gobalMgoSession = session
}

func (p *FollowerHandlerSuite) cleanTestDataIfExist(c *C, collection *mgo.Collection, userId string) {
	err := collection.Remove(bson.M{"userId": userId})
	if err != nil && err != mgo.ErrNotFound {
		c.Fatal(err)
	}
}

func (p *FollowerHandlerSuite) CreateTestDoc(userId string, coins int64, orderIds []string) bson.M {
	doc := bson.M{}

	doc["userId"] = userId
	doc["coins"] = coins
	doc["lastPushDate"] = int64(0)

	var orders []bson.M
	for j := 0; j < len(orderIds); j++ {
		order := bson.M{}
		order["orderId"] = orderIds[j]
		order["date"] = int64(time.Now().AddDate(0, 0, rand.Int()%30).Unix())
		order["coins"] = int64(j * 10)
		order["fans"] = order["coins"].(int64) * 100
		if j%2 == 0 {
			order["progress"] = order["fans"].(int64) - 1
		} else {
			order["progress"] = order["fans"].(int64)
		}

		if j%2 == 0 {
			order["status"] = false
		} else {
			order["status"] = true
		}

		orders = append(orders, order)
	}

	doc["orders"] = orders

	return doc
}

func (p *FollowerHandlerSuite) TearDownTest(c *C) {
	session, err := mgo.Dial(testGobalMongoUri)
	if err != nil {
		c.Fatal(err)
	}

	collection := session.DB(testGobalDbName).C(testGobalCollName)
	p.cleanTestDataIfExist(c, collection, p.userId)
}

func (p *FollowerHandlerSuite) Test_counterHander(c *C) {

}

func (p *FollowerHandlerSuite) Test_coinsHandler(c *C) {
	incCoins := int64(10)
	url := fmt.Sprintf("https://%v/getfollowers/coins?userId=%v&version=%v&coins=%v", testGobalHttpAddr, p.userId, testGobalVersion, incCoins)

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	coinsHandler(w, request)
	if w.Code != 200 {
		c.Fatal(w.Body.String())
	}

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		c.Fatal(err)
	}

	c.Assert(int64(result["coins"].(float64)), Equals, p.Coins+incCoins)
	c.Assert(result["userId"].(string), Equals, p.userId)
}

func (p *FollowerHandlerSuite) Test_coinsHander_userIdNotExist(c *C) {
	collection := p.session.DB(testGobalDbName).C(testGobalCollName)
	p.cleanTestDataIfExist(c, collection, testGobalUserIdNotExist)

	url := fmt.Sprintf("https://%v/getfollowers/coins?userId=%v&version=%v&coins=%v", testGobalHttpAddr, testGobalUserIdNotExist, testGobalVersion, 0)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_USER_NOT_FOUND)
		} else {
			c.Fatal("not FollowerError")
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()
	coinsHandler(recorder, request)
	c.Fatal("no error found")
}

func (p *FollowerHandlerSuite) Test_coinsHander_invalidUrl_noVersion(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/coins?userId=%v&coins=%v", testGobalHttpAddr, p.userId, 0)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_URL_PARAM_INVALID)
		} else {
			c.Fatal("not FollowerError")
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()
	coinsHandler(recorder, request)
	c.Fatal("no error found")
}

func (p *FollowerHandlerSuite) Test_coinsHander_invalidUrl_noUserId(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/coins?version=%v&coins=%v", testGobalHttpAddr, testGobalVersion, 0)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_URL_PARAM_INVALID)
		} else {
			c.Fatal("not FollowerError")
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()
	coinsHandler(recorder, request)
	c.Fatal("no error found")
}

func (p *FollowerHandlerSuite) Test_coinsHander_invalidUrl_noCoins(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/coins?version=%v&userId=%v", testGobalHttpAddr, testGobalVersion, p.userId)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_URL_PARAM_INVALID)
		} else {
			c.Fatal("not FollowerError")
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()
	coinsHandler(recorder, request)
	c.Fatal("no error found")
}

func (p *FollowerHandlerSuite) Test_infoHandler(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/info?userId=%v&version=%v", testGobalHttpAddr, p.userId, testGobalVersion)

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	infoHandler(w, request)
	if w.Code != 200 {
		c.Fatal(w.Body.String())
	}

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		c.Fatal(err)
	}

	c.Assert(result["userId"].(string), Equals, p.userId)
	c.Assert(int64(result["coins"].(float64)), Equals, p.Coins)
}

func (p *FollowerHandlerSuite) Test_infoHandler_userIdNotExist(c *C) {
	collection := p.session.DB(testGobalDbName).C(testGobalCollName)

	p.cleanTestDataIfExist(c, collection, testGobalUserIdNotExist)

	url := fmt.Sprintf("https://%v/getfollowers/info?userId=%v&version=%v", testGobalHttpAddr, testGobalUserIdNotExist, testGobalVersion)

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	infoHandler(w, request)
	if w.Code != 200 {
		c.Fatal(w.Body.String())
	}

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		c.Fatal(err)
	}

	c.Assert(result["userId"].(string), Equals, testGobalUserIdNotExist)
	c.Assert(int64(result["coins"].(float64)), Equals, int64(0))

	p.cleanTestDataIfExist(c, collection, testGobalUserIdNotExist)
}

func (p *FollowerHandlerSuite) Test_infoHandler_invalidUrl_noUserId(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/info?version=%v", testGobalHttpAddr, testGobalVersion)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_URL_PARAM_INVALID)
		} else {
			c.Fatal(err)
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	infoHandler(w, request)

	c.Fatal("no error")
}

func (p *FollowerHandlerSuite) Test_infoHandler_invalidUrl_noVersion(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/info?userId=%v", testGobalHttpAddr, p.userId)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_URL_PARAM_INVALID)
		} else {
			c.Fatal(err)
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	infoHandler(w, request)

	c.Fatal("no error")
}

func (p *FollowerHandlerSuite) Test_progressHandler(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/progress?userId=%v&version=%v", testGobalHttpAddr, p.userId, testGobalVersion)

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	infoHandler(w, request)
	if w.Code != 200 {
		c.Fatal(w.Body.String())
	}

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		c.Fatal(err)
	}

	c.Assert(result["userId"].(string), Equals, p.userId)
	c.Assert(int64(result["coins"].(float64)), Equals, p.Coins)
	orders := result["orders"].([]interface{})
	c.Assert(len(orders), Equals, p.OrderCount)
}

func (p *FollowerHandlerSuite) Test_progressHandler_userIdNotExist(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/progress?userId=%v&version=%v", testGobalHttpAddr, testGobalUserIdNotExist, testGobalVersion)

	collection := p.session.DB(testGobalDbName).C(testGobalCollName)
	p.cleanTestDataIfExist(c, collection, testGobalUserIdNotExist)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_USER_NOT_FOUND)
		} else {
			c.Fatal("not FollowerError")
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	progressHandler(w, request)
	c.Fatal("no error found")
}

func (p *FollowerHandlerSuite) Test_progressHandler_invalidUrl_noUserId(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/progress?version=%v", testGobalHttpAddr, testGobalVersion)

	collection := p.session.DB(testGobalDbName).C(testGobalCollName)
	p.cleanTestDataIfExist(c, collection, testGobalUserIdNotExist)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_URL_PARAM_INVALID)
		} else {
			c.Fatal("not FollowerError")
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	progressHandler(w, request)
	c.Fatal("no error found")
}

func (p *FollowerHandlerSuite) Test_progressHandler_invalidUrl_noVersion(c *C) {
	url := fmt.Sprintf("https://%v/getfollowers/progress?userId=%v", testGobalHttpAddr, p.userId)

	collection := p.session.DB(testGobalDbName).C(testGobalCollName)
	p.cleanTestDataIfExist(c, collection, testGobalUserIdNotExist)

	defer func() {
		if err, ok := recover().(FollowerError); ok {
			c.Assert(err.Code, Equals, ERROR_URL_PARAM_INVALID)
		} else {
			c.Fatal("not FollowerError")
		}
	}()

	request := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	progressHandler(w, request)
	c.Fatal("no error found")
}
