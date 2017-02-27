package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func progressHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	err := validProgressUrlParam(r.Form)
	checkError(err)

	result, err := queryProgress(r.Form)
	checkError(err)

	responseToClient(w, result)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	checkError(validInfoUrlParam(r.Form))
	result, err := queryInfo(r.Form)
	checkError(err)

	responseToClient(w, result)
}

func getuserHandler(w http.ResponseWriter, r *http.Request) {

}

func buyfollowerHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	checkError(validBuyFollowerUrlParam(r.Form))

	userid := r.Form["userid"][0]
	coins := r.Form["coins"][0]
	coinsInt, _ := strconv.Atoi(coins)
	fans := r.Form["value"][0]
	fansInt, _ := strconv.Atoi(fans)

	order := Order{
		OrderId:  fmt.Sprintf("%v", uuid.NewV4()),
		Date:     time.Now().Unix(),
		Coins:    int64(coinsInt),
		Fans:     int64(fansInt),
		Progress: 0,
		Status:   false,
	}

	orders := make([]Order, 0, 1)
	orders = append(orders, order)

	buyer := Buyer{
		UserId:     userid,
		ClickCoins: 0,
		Orders:     orders,
	}

	checkError(g_buyerManager.Add(buyer))
}

func coinsHandler(w http.ResponseWriter, r *http.Request) {

}

func queryProgress(values url.Values) (bson.M, error) {
	session := g_mgoSession.Copy()
	defer session.Close()

	collection := session.DB("follower").C("user")
	queryStatement := bson.M{"userid": values["userid"][0]}
	query := collection.Find(queryStatement)

	var result bson.M
	err := query.One(&result)
	if err != nil {
		err = NewError("[queryProgress] query.one failed. error=%v", err.Error())
		return nil, err
	}

	return result, nil
}

func queryInfo(values url.Values) (bson.M, error) {
	session := g_mgoSession.Copy()
	defer session.Close()

	collection := session.DB("follower").C("user")

	userid := values["userid"][0]
	q := bson.M{"userid": userid}
	query := collection.Find(q)

	var result bson.M
	err := query.One(&result)
	if err != nil {
		err = NewError("[queryInfo] query.one failed. error=%v", err.Error())
		return nil, err
	}

	return result, nil
}

func responseToClient(w http.ResponseWriter, info interface{}) {
	w.WriteHeader(200)

	respByte, err := json.Marshal(info)
	if err != nil {
		log.Error(fmt.Sprintf("[responseToClient] json.Marshal failed. error=%v", err))
		return
	}
	_, err = io.WriteString(w, string(respByte))
	if err != nil {
		log.Error(fmt.Sprintf("[responseToClient] io.WriteString failed. error=%v", err))
		return
	}
}

func validBuyFollowerUrlParam(values url.Values) error {
	id := values["userid"][0]
	if len(id) == 0 {
		return errors.New("[validBuyFollowerUrlParam] len(id) == 0.")
	}

	version := values["version"][0]
	if !validVersion(version) {
		return errors.New(fmt.Sprintf("[validBuyFollowerUrlParam] version invalid. version=%v", version))
	}

	coins := values["coins"][0]
	coinsInt, err := strconv.Atoi(coins)
	if err != nil || coinsInt == 0 {
		return errors.New(fmt.Sprintf("[validBuyFollowerUrlParam] coins invalid. conins:%v", coins))
	}

	followers := values["value"][0]
	followersInt, err := strconv.Atoi(followers)
	if err != nil || followersInt == 0 {
		return errors.New(fmt.Sprintf("[validBuyFollowerUrlParam] followers count invalid. count:%v", followers))
	}

	return nil
}

func validProgressUrlParam(values url.Values) error {
	id := values["userid"][0]
	if len(id) == 0 {
		return errors.New("[validInfoUrlParam] len(id) == 0.")
	}

	version := values["version"][0]
	if !validVersion(version) {
		return errors.New(fmt.Sprintf("[validInfoUrlParaml] version invalid. version=%v", version))
	}

	return nil
}

func validInfoUrlParam(values url.Values) error {
	id := values["userid"][0]
	if len(id) == 0 {
		return errors.New("[validInfoUrlParam] len(id) == 0.")
	}

	version := values["version"][0]
	if !validVersion(version) {
		return errors.New(fmt.Sprintf("[validInfoUrlParaml] version invalid. version=%v", version))
	}

	return nil
}

func validVersion(v string) bool {
	intV, err := strconv.Atoi(v)
	if err != nil {
		log.Error("[validVersion] strconv.atoi failed. version=%v", v)
		return false
	}

	switch intV {
	case 1:
		return true
	default:
		return false
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewError(format string, a ...interface{}) error {
	s := fmt.Sprintf(format, a)
	return errors.New(s)
}
