package orders

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qor/transition"
)

var (
	// OrderState order's state machine
	OrderState = transition.New(&Order{})

	// ItemState order item's state machine
	ItemState = transition.New(&OrderItem{})
)

var (
	// DraftState draft state
	DraftState = "draft"
)

func init() {
	// Define Order's States
	OrderState.Initial("draft")
	OrderState.State("checkout")
	OrderState.State("cancelled").Enter(func(value interface{}, tx *gorm.DB) error {
		tx.Model(value).UpdateColumn("cancelled_at", time.Now())
		return nil
	})
	OrderState.State("paid").Enter(func(value interface{}, tx *gorm.DB) (err error) {
		var orderItems []OrderItem

		tx.Model(value).Association("OrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err = ItemState.Trigger("pay", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		tx.Save(value)
		// freeze stock, change items's state
		return nil
	})
	OrderState.State("paid_cancelled").Enter(func(value interface{}, tx *gorm.DB) error {
		// do refund, release stock, change items's state
		return nil
	})
	OrderState.State("processing").Enter(func(value interface{}, tx *gorm.DB) (err error) {
		var orderItems []OrderItem
		tx.Model(value).Association("OrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err = ItemState.Trigger("process", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	OrderState.State("shipped").Enter(func(value interface{}, tx *gorm.DB) (err error) {
		tx.Model(value).UpdateColumn("shipped_at", time.Now())

		var orderItems []OrderItem
		tx.Model(value).Association("OrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err = ItemState.Trigger("ship", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	OrderState.State("returned")

	OrderState.Event("checkout").To("checkout").From("draft")
	OrderState.Event("pay").To("paid").From("checkout")
	cancelEvent := OrderState.Event("cancel")
	cancelEvent.To("cancelled").From("draft", "checkout")
	cancelEvent.To("paid_cacelled").From("paid", "processing", "shipped")
	OrderState.Event("process").To("processing").From("paid")
	OrderState.Event("ship").To("shipped").From("processing")
	OrderState.Event("return").To("returned").From("shipped")

	// Define ItemItem's States
	ItemState.Initial("checkout")
	ItemState.State("cancelled").Enter(func(value interface{}, tx *gorm.DB) error {
		// release stock, upate order state
		return nil
	})
	ItemState.State("paid").Enter(func(value interface{}, tx *gorm.DB) error {
		// freeze stock, update order state
		return nil
	})
	ItemState.State("paid_cancelled").Enter(func(value interface{}, tx *gorm.DB) error {
		// do refund, release stock, update order state
		return nil
	})
	ItemState.State("processing")
	ItemState.State("shipped")
	ItemState.State("returned")

	ItemState.Event("checkout").To("checkout").From("draft")
	ItemState.Event("pay").To("paid").From("checkout")
	cancelItemEvent := ItemState.Event("cancel")
	cancelItemEvent.To("cancelled").From("checkout")
	cancelItemEvent.To("paid_cancelled").From("paid")
	ItemState.Event("process").To("processing").From("paid")
	ItemState.Event("ship").To("shipped").From("processing")
	ItemState.Event("return").To("returned").From("shipped")
}