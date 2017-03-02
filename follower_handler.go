package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

	userId := r.Form["userId"][0]
	coins := r.Form["coins"][0]
	coinsInt, _ := strconv.Atoi(coins)
	fans := r.Form["value"][0]
	fansInt, _ := strconv.Atoi(fans)
	orderId := r.Form["orderId"][0]

	order := Order{
		orderId,
		time.Now().Unix(),
		int64(coinsInt),
		int64(fansInt),
		0, false,
	}

	session := g_mgoSession.Copy()
	collection := session.DB("follower").C("user")

	query := collection.Find(bson.M{"userId": userId})
	var result bson.M
	err := query.One(&result)

	if err != nil {
		if err.Error() != "not found" {
			err = NewError("[buyfollowerHandler] query.one failed. error=%v", err)
			checkError(err)
		} else {
			doc := bson.M{
				"userId":     userId,
				"clickCoins": 0,
				"orders":     order,
			}

			err := collection.Insert(doc)
			if err != nil {
				err = NewError("[buyfollowerHandler] collection.insert failed. error=%v", err)
			}
			checkError(err)

			item := PushItem{&order, userId}
			g_pushManger.Add(&item)

			responseToClient(w, nil)
		}
	} else {
		orderAlreadyExist := false
		if orders, ok := result["orders"]; ok {
			for _, order := range orders.([]map[string]interface{}) {
				if orderId == order["orderId"].(string) {
					orderAlreadyExist = true
					break
				}
			}
		}

		if orderAlreadyExist {
			responseToClient(w, nil)
		} else {
			doc := bson.M{
				"userId":     userId,
				"clickCoins": 0,
				"orders": []bson.M{
					bson.M{
						"orderId":  orderId,
						"date":     time.Now().Unix(),
						"coins":    int64(coinsInt),
						"fans":     int64(fansInt),
						"progress": 0,
						"status":   false,
					},
				},
			}

			err := collection.Update(bson.M{"userId": userId}, bson.M{"$push": bson.M{"orders": doc}})
			if err != nil {
				err = NewError("[buyfollowerHandler] collection.update failed. error=%v", err)
			}
			checkError(err)

			item := PushItem{&order, userId}
			g_pushManger.Add(&item)

			responseToClient(w, nil)
		}
	}
}

func coinsHandler(w http.ResponseWriter, r *http.Request) {

}

func queryProgress(values url.Values) (bson.M, error) {
	session := g_mgoSession.Copy()
	defer session.Close()

	collection := session.DB("follower").C("user")
	queryStatement := bson.M{"userId": values["userId"][0]}
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

	userId := values["userId"][0]
	q := bson.M{"userId": userId}
	query := collection.Find(q)

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

	orderId := values["orderId"][0]
	if len(orderId) != 36 {
		return NewError("[validBuyFollowerUrlParam] orederid invalid. orderId:%v", orderId)
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
