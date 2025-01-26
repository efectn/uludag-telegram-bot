package database

import (
	"encoding/json"
	"errors"

	"go.etcd.io/bbolt"
)

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ChatID              string `json:"chat_id"`
	StudentID           string `json:"student_id"`
	StudentSessionToken string `json:"student_session_token"`
}

type Database struct {
	db *bbolt.DB
}

func NewDatabase(location string) (*Database, error) {
	instance, err := bbolt.Open(location, 0600, nil)
	if err != nil {
		return nil, err
	}

	db := &Database{}
	db.db = instance

	// Create users bucket
	err = db.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}
		return nil
	})

	return db, nil
}

func (d *Database) SaveUser(user User) error {
	err := d.db.Update(func(tx *bbolt.Tx) error {
		encoded, err := json.Marshal(user)
		if err != nil {
			return err
		}

		bucket := tx.Bucket([]byte("users"))
		err = bucket.Put([]byte(user.ChatID), encoded)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (d *Database) GetUser(chatID string) (User, error) {
	var user User

	err := d.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		data := bucket.Get([]byte(chatID))
		if data == nil {
			return ErrUserNotFound
		}

		err := json.Unmarshal(data, &user)
		if err != nil {
			return err
		}

		return nil
	})

	return user, err
}

func (d *Database) AllUsers() ([]User, error) {
	var users []User

	err := d.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		err := bucket.ForEach(func(k, v []byte) error {
			var user User
			err := json.Unmarshal(v, &user)
			if err != nil {
				return err
			}

			users = append(users, user)
			return nil
		})

		return err
	})

	return users, err
}

func (d *Database) DeleteUser(chatID string) error {
	err := d.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		err := bucket.Delete([]byte(chatID))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}
