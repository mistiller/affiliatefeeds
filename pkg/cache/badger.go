package cache

import (
	"time"

	badger "github.com/dgraph-io/badger"
	log "github.com/sirupsen/logrus"
	zip "stillgrove.com/gofeedyourself/pkg/zip"
)

type BadgerCache struct {
	db  *badger.DB
	ttl time.Duration
}

// NewBadgerCache returns a Cache, takes path to cache file on disk (creates file if neccessary)
func NewBadgerCache(file string, ttl time.Duration) (c Cache, err error) {
	l := log.New()
	l.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	l.SetLevel(log.WarnLevel)
	db, err := badger.Open(badger.DefaultOptions(file).WithLogger(l))
	if err != nil {
		return c, err
	}
	return BadgerCache{
		db:  db,
		ttl: ttl,
	}, nil
}

func (b BadgerCache) Load(key string) (payload []byte, err error) {
	var zipped []byte
	err = b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			return nil
		})
		if err != nil {
			return err
		}

		zipped, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})

	payload, err = zip.Unzip(zipped)
	if err != nil {
		return nil, err
	}

	return payload, err
}

func (b BadgerCache) Store(updates map[string][]byte) (err error) {
	var payload []byte
	txn := b.db.NewTransaction(true)
	for k, v := range updates {
		payload, err = zip.Zip(v)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(k), payload).WithTTL(b.ttl)
		if err := txn.SetEntry(e); err == badger.ErrTxnTooBig {
			_ = txn.Commit()
			txn = b.db.NewTransaction(true)
			_ = txn.SetEntry(e)
		}
	}
	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (b BadgerCache) LoadAll() (outputs map[string][]byte, err error) {
	outputs = make(map[string][]byte)
	err = b.db.View(func(txn *badger.Txn) error {
		var k []byte
		var item *badger.Item

		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item = it.Item()
			k = item.Key()
			err := item.Value(func(v []byte) error {
				v, _ = zip.Unzip(v)
				outputs[string(k)] = v
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return outputs, err
}

func (b BadgerCache) Close() {
	b.db.Close()
}
