package badger

import (
	"context"
	"fmt"
	"sort"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"

	"github.com/derfenix/webarchive/adapters/repository"
	"github.com/derfenix/webarchive/entity"
)

func NewCollection(db *badger.DB) (*Collection, error) {
	return &Collection{
		db:     db,
		prefix: []byte("collection:"),
	}, nil
}

type Collection struct {
	db     *badger.DB
	prefix []byte
}

func (c *Collection) Save(_ context.Context, col *entity.Collection) error {
	if c.db.IsClosed() {
		return repository.ErrDBClosed
	}

	marshaled, err := marshal(col)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	if err := c.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(c.key(col), marshaled); err != nil {
			return fmt.Errorf("put data: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update db: %w", err)
	}

	return nil
}

func (c *Collection) Delete(_ context.Context, id uuid.UUID) error {
	if c.db.IsClosed() {
		return repository.ErrDBClosed
	}

	col := entity.Collection{ID: id}
	key := c.key(&col)

	if err := c.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(key); err != nil {
			return fmt.Errorf("delete data: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update db: %w", err)
	}

	return nil
}

func (c *Collection) Get(_ context.Context, id uuid.UUID) (*entity.Collection, error) {
	col := entity.Collection{ID: id}

	err := c.db.View(func(txn *badger.Txn) error {
		data, err := txn.Get(c.key(&col))
		if err != nil {
			return fmt.Errorf("get data: %w", err)
		}

		err = data.Value(func(val []byte) error {
			if err := unmarshal(val, &col); err != nil {
				return fmt.Errorf("unmarshal data: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("get value: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("view: %w", err)
	}

	return &col, nil
}

func (c *Collection) ListAll(ctx context.Context, owner string) ([]*entity.Collection, error) {
	cols := make([]*entity.Collection, 0, 100)

	err := c.db.View(func(txn *badger.Txn) error {
		iterator := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iterator.Close()

		for iterator.Seek(c.prefix); iterator.ValidForPrefix(c.prefix); iterator.Next() {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context canceled: %w", err)
			}

			var col entity.Collection
			err := iterator.Item().Value(func(val []byte) error {
				if err := unmarshal(val, &col); err != nil {
					return fmt.Errorf("unmarshal: %w", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("get item: %w", err)
			}

			if col.Owner != owner {
				continue
			}

			cols = append(cols, &col)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("view: %w", err)
	}

	sort.Slice(cols, func(i, j int) bool {
		return cols[i].Created.After(cols[j].Created)
	})

	return cols, nil
}

func (c *Collection) key(col *entity.Collection) []byte {
	return append(c.prefix, []byte(col.ID.String())...)
}
