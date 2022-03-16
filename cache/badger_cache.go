package cache

import (
	"time"

	"github.com/dgraph-io/badger/v3"
)

type BadgerCache struct {
	Conn   *badger.DB
	Prefix string
}

func (c *BadgerCache) Has(str string) (bool, error) {
	_, err := c.Get(str)

	if err != nil {
		return false, nil
	}

	return true, nil
}

func (c *BadgerCache) Get(str string) (interface{}, error) {
	var fromCache []byte

	err := c.Conn.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(str))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			fromCache = append([]byte{}, val...)
			return nil
		})

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	decoded, err := decode(string(fromCache))
	if err != nil {
		return nil, err
	}

	item := decoded[str]

	return item, nil
}

func (c *BadgerCache) Set(str string, value interface{}, expires ...int) error {
	entry := Entry{}

	entry[str] = value
	encoded, err := encode(entry)
	if err != nil {
		return err
	}

	if len(expires) > 0 {
		err = c.Conn.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(str), encoded).WithTTL(time.Second * time.Duration(expires[0]))
			err = txn.SetEntry(e)
			return err
		})
	} else {
		err = c.Conn.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(str), encoded)
			err = txn.SetEntry(e)
			return err
		})
	}

	return nil
}

func (c *BadgerCache) Forget(str string) error {
	err := c.Conn.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(str))
		return err
	})

	return err
}

func (c *BadgerCache) EmptyByMatch(str string) error {
	return c.emptyByMatch(str)
}

func (c *BadgerCache) Empty() error {
	return c.emptyByMatch("")
}

func (c *BadgerCache) emptyByMatch(str string) error {
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := c.Conn.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 100000

	err := c.Conn.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.AllVersions = false
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0

		for it.Seek([]byte(str)); it.ValidForPrefix([]byte(str)); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					return err
				}
			}

		}

		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
