package main

import (
	"fmt"
	"github.com/petar/GoLLRB/llrb"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"sync"
)

type Order struct {
	OrderId  string `bson:"orderId"`
	Date     int64  `bson:"date"`
	Coins    int64  `bson:"coins"`
	Fans     int64  `bson:"fans"`
	Progress int64  `bson:"progress"`
	Status   bool   `bson:"status"`
}

type PushItem struct {
	Order  *Order
	UserId string
}

func (p PushItem) Less(than llrb.Item) bool {
	if p.Order.Date < than.(*PushItem).Order.Date {
		return true
	} else if p.Order.Date > than.(*PushItem).Order.Date {
		return false
	} else {
		return p.UserId != than.(*PushItem).UserId
	}
}

type PushManager struct {
	mutex sync.Mutex
	items llrb.LLRB
}

var gobalPushManger = PushManager{}

func (p *PushManager) Add(item *PushItem) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.items.InsertNoReplace(item)
}

func (p *PushManager) push(w http.ResponseWriter, userId string, num int) error {
	session := gobalMgoSession.Copy()
	collection := session.DB("follower").C("user")
	query := collection.Find(bson.M{"userId": userId}).Select(bson.M{"_id": 0, "lastPushDate": 1})
	var result bson.M
	err := query.One(&result)
	if err != nil {
		return NewError(ERROR_DB_OPERATE_FAIELD, "[PushManager.push] query.one failed. error=%v", err)
	}

	lastPushDate := result["lastPushDate"].(int64)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	order := &Order{Date: lastPushDate}
	item := &PushItem{Order: order, UserId: userId}
	pushList := make([]*PushItem, 0, num)
	p.items.AscendGreaterOrEqual(item, func(i llrb.Item) bool {
		fmt.Println(i.(*PushItem).Order.OrderId)
		if i.(*PushItem).Order.Date == lastPushDate && i.(*PushItem).UserId == userId {
			return true
		}
		pushList = append(pushList, i.(*PushItem))
		if len(pushList) == num {
			return false
		}

		return true
	})

	if len(pushList) == 0 {
		return NewError(ERROR_NO_BUYER, "[PushManager.push] no buyer")
	}

	userIDs := make([]string, 0, len(pushList))
	for _, v := range pushList {
		userIDs = append(userIDs, v.UserId)
	}

	err = responseToClient(w, bson.M{"userIDs": userIDs})
	if err != nil {
		return err
	}

	var s bson.M
	var u bson.M
	updatePairs := make([]interface{}, 0, len(pushList)*2+1)

	for _, item := range pushList {
		fmt.Println(item.Order.Fans)
		fmt.Println(item.Order.Progress)

		if item.Order.Progress+1 == item.Order.Fans {
			u = bson.M{"$set": bson.M{"orders.$.progress": item.Order.Progress + 1, "orders.$.status": true}}

		} else {
			u = bson.M{"$set": bson.M{"orders.$.progress": item.Order.Progress + 1}}

		}

		s = bson.M{"orders.orderId": item.Order.OrderId}
		updatePairs = append(updatePairs, s)
		updatePairs = append(updatePairs, u)
	}

	s = bson.M{"userId": userId}
	u = bson.M{"$set": bson.M{"lastPushDate": pushList[len(pushList)-1].Order.Date}}
	updatePairs = append(updatePairs, s)
	updatePairs = append(updatePairs, u)

	bulk := collection.Bulk()
	for i := 0; i < len(updatePairs); i += 2 {
		bulk.Update(updatePairs[i], updatePairs[i+1])
	}

	_, err = bulk.Run()
	if err != nil {
		return NewError(ERROR_DB_OPERATE_FAIELD, "[PushManager.push] bulk.run failed. error=%v", err)
	}

	for _, v := range pushList {
		if v.Order.Progress+1 == v.Order.Fans {
			p.delPushItem(v)
		} else {
			v.Order.Progress += 1
		}
	}

	return nil
}

func (p *PushManager) GetPushItems(key PushItem, itemNum int) []*PushItem {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	result := make([]*PushItem, 0, itemNum)
	p.items.AscendGreaterOrEqual(key, func(i llrb.Item) bool {
		if len(result) != itemNum {
			result = append(result, i.(*PushItem))
			return true
		} else {
			return false
		}
	})

	return result
}

func (p *PushManager) delPushItem(key *PushItem) {
	defer p.mutex.Unlock()
}
