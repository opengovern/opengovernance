package db

import "github.com/kaytu-io/kaytu-engine/services/subscription/db/model"

func (db Database) CreateSubscription(sub *model.Subscription) error {
	return db.Orm.Model(&model.Subscription{}).Create(sub).Error
}

func (db Database) ListSubscriptions() ([]model.Subscription, error) {
	var subs []model.Subscription
	err := db.Orm.Model(&model.Subscription{}).Find(&subs).Error
	return subs, err
}
