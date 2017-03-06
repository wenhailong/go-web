package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2"
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

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	checkError(validUserUrlParam(r.Form))

	userId := r.Form["userId"][0]
	checkError(g_pushManger.push(w, userId, 2))
}

func validUserUrlParam(values url.Values) error {
	id := values["userId"][0]
	if len(id) == 0 {
		return NewError("[validBuyFollowerUrlParam] len(id) == 0.")
	}

	version := values["version"][0]
	if !validVersion(version) {
		return NewError("[validBuyFollowerUrlParam] version invalid. version=%v", version)
	}

	return nil
}

func buyfollowerHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	checkError(validBuyFollowerUrlParam(r.Form))

	userId := r.Form["userId"][0]
	coins := r.Form["coins"][0]
	coinsInt, _ := strconv.Atoi(coins)
	fans := r.Form["value"][0]
	fansInt, _ := strconv.Atoi(fans)

	order := Order{
		fmt.Sprintf("%v", uuid.NewV4()),
		time.Now().Unix(),
		int64(coinsInt),
		int64(fansInt),
		0, false,
	}

	session := g_mgoSession.Copy()
	collection := session.DB("follower").C("user")

	var result bson.M
	queryStatement := bson.M{"userId": userId, "coins": bson.M{"$gte": coinsInt}}
	query := collection.Find(queryStatement)
	change := mgo.Change{Update: bson.M{"$inc": bson.M{"coins": -coinsInt}, "$addToSet": bson.M{"orders": order}}, ReturnNew: true}
	_, err := query.Apply(change, &result)
	if err != nil {
		err = NewError("[buyfollowerHandler] query.Apply failed. error=%v", err)
	}
	checkError(err)

	item := PushItem{&order, userId}
	g_pushManger.Add(&item)

	respInfo := bson.M{"coins": result["coins"]}
	responseToClient(w, respInfo)
}

func coinsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	checkError(validCoinsUrlParam(r.Form))

	session := g_mgoSession.Copy()
	defer session.Close()

	collection := session.DB("follower").C("user")

	userId := r.Form["userId"][0]
	coins := r.Form["coins"][0]
	coinsInt, _ := strconv.Atoi(coins)

	queryStatement := bson.M{"userId": userId}
	change := mgo.Change{Update: bson.M{"$inc": bson.M{"coins": coinsInt}}, ReturnNew: true}

	var result bson.M
	_, err := collection.Find(queryStatement).Apply(change, &result)
	if err != nil {
		err = NewError("[coinsHandler] collection.Apply failed. error=%v", err)
	}
	checkError(err)

	delete(result, "_id")
	delete(result, "orders")

	responseToClient(w, result)
}

func validCoinsUrlParam(values url.Values) error {
	id := values["userId"][0]
	if len(id) == 0 {
		return errors.New("[validInfoUrlParam] len(id) == 0.")
	}

	version := values["version"][0]
	if !validVersion(version) {
		return errors.New(fmt.Sprintf("[validInfoUrlParaml] version invalid. version=%v", version))
	}

	coins := values["coins"][0]
	_, err := strconv.Atoi(coins)
	if err != nil {
		return errors.New(fmt.Sprintf("[validInfoUrlParaml] coins invalid. coins=%v", coins))
	}

	return nil
}

func queryProgress(values url.Values) (bson.M, error) {
	session := g_mgoSession.Copy()
	defer session.Close()

	collection := session.DB("follower").C("user")
	queryStatement := bson.M{"userId": values["userId"][0]}
	selectorStatement := bson.M{"_id": 0, "userId": 1, "orders.fans": 1, "orders.progress": 1, "orders.status": 1}
	query := collection.Find(queryStatement).Select(selectorStatement)

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

	userId := values["userId"][0]
	queryStatemnt := bson.M{"userId": userId}
	selectorStatement := bson.M{"_id": 0, "userId": 1, "coins": 1, "orders.fans": 1, "orders.progress": 1, "orders.status": 1}
	query := collection.Find(queryStatemnt).Select(selectorStatement)

	var result bson.M
	err := query.One(&result)
	if err != nil {
		err = NewError("[queryInfo] query.one failed. error=%v", err.Error())
		return nil, err
	}

	return result, nil
}

func responseToClient(w http.ResponseWriter, info interface{}) error {
	w.WriteHeader(200)

	respByte, err := json.Marshal(info)
	if err != nil {
		return NewError(fmt.Sprintf("[responseToClient] json.Marshal failed. error=%v", err))
	}
	_, err = io.WriteString(w, string(respByte))
	if err != nil {
		return NewError(fmt.Sprintf("[responseToClient] io.WriteString failed. error=%v", err))
	}

	return nil
}

func validBuyFollowerUrlParam(values url.Values) error {
	id := values["userId"][0]
	if len(id) == 0 {
		return NewError("[validBuyFollowerUrlParam] len(id) == 0.")
	}

	version := values["version"][0]
	if !validVersion(version) {
		return NewError("[validBuyFollowerUrlParam] version invalid. version=%v", version)
	}

	coins := values["coins"][0]
	coinsInt, err := strconv.Atoi(coins)
	if err != nil || coinsInt == 0 {
		return NewError("[validBuyFollowerUrlParam] coins invalid. conins:%v", coins)
	}

	followers := values["value"][0]
	followersInt, err := strconv.Atoi(followers)
	if err != nil || followersInt == 0 {
		return NewError("[validBuyFollowerUrlParam] followers count invalid. count:%v", followers)
	}

	return nil
}

func validProgressUrlParam(values url.Values) error {
	id := values["userId"][0]
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
	id := values["userId"][0]
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
		return false
	}

	switch intV {
	case 1:
		return true
	default:
		return false
	}
}
