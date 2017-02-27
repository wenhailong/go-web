package main

import (
	"errors"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

type Order struct {
	OrderId  string `bson:"orderid"`
	Date     int64  `bson:"date"`
	Coins    int64  `bson:"coins"`
	Fans     int64  `bson:"fans"`
	Progress int    `bson:"progress"`
	Status   bool   `bson:"status"`
}

type Buyer struct {
	ObjectId   bson.ObjectId `bson:"_id"`
	UserId     string        `bson:"userid"`
	ClickCoins int           `bson:"clickCoins"`
	Orders     []Order       `bson:"orders"`
}

type BuyerManager map[string]Buyer

var buyerManagerMutex sync.Mutex

func NewBuyerManager() BuyerManager {
	return make(map[string]Buyer)
}

var g_buyerManager = NewBuyerManager()

func (p *BuyerManager) Add(newBuyer Buyer) error {
	id := newBuyer.UserId

	buyerManagerMutex.Lock()
	defer buyerManagerMutex.Unlock()

	if buyer, ok := (*p)[id]; ok {
		newOrders := newBuyer.Orders
		err := updateBuyerOrdersToDB(id, newOrders)
		if err != nil {
			return err
		}

		appendNewOrder(buyer, newOrders)

	} else {
		err := insertBuyerToDB(newBuyer)
		if err != nil {
			return err
		}

		p.updateBuyers(newBuyer)
	}

	return nil
}

func (p *BuyerManager) updateBuyers(buyer Buyer) {
	(*p)[buyer.UserId] = buyer
}

func updateBuyerOrdersToDB(userid string, orders []Order) error {
	session := g_mgoSession.Copy()
	defer session.Close()

	collection := session.DB("follower").C("user")

	selecotr := bson.M{"userid": userid}
	update := bson.M{"$pushAll": bson.M{"orders": orders}}
	err := collection.Update(selecotr, update)
	if err != nil {
		errors.New(fmt.Sprintf("[updateBuyerOrdersToDB] update db failed. selector=%v update=%v", selecotr, update))
	}

	return nil
}

func insertBuyerToDB(newBuyer Buyer) error {
	session := g_mgoSession.Copy()
	defer session.Close()

	collection := session.DB("follower").C("user")
	err := collection.Insert(newBuyer)

	if err != nil {
		return errors.New(fmt.Sprintf("[insertBuyerToDB] add buyer failed. err=%c", err))
	}

	return nil
}

func appendNewOrder(buyer Buyer, orders []Order) {
	oldOrders := buyer.Orders

	for _, order := range orders {
		oldOrders = append(oldOrders, order)
	}

	buyer.Orders = oldOrders
}
