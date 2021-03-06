package cimp

import (
	"fmt"

	"github.com/hashicorp/consul/api"
)

type Config struct {
	Address string
}

type ConsulStorage struct {
	client *api.Client
}

const consulTransactionLimit = 64

func NewStorage(cfg Config) (*ConsulStorage, error) {
	clientCfg := api.DefaultConfig()
	clientCfg.Address = cfg.Address

	client, err := api.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("create consul client: %w", err)
	}

	return &ConsulStorage{
		client: client,
	}, nil
}

func (cs *ConsulStorage) Save(kv *KV) error {
	var ops api.TxnOps

	var i int
	for key, path := range kv.idx {
		i++
		leaf, err := kv.tree.Get(path)
		if err != nil {
			return fmt.Errorf("get key %q value from tree: %w", key, err)
		}

		op := &api.TxnOp{
			KV: &api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   kv.globalPrefix + key,
				Value: []byte(fmt.Sprint(leaf.Value)),
			},
		}
		ops = append(ops, op)

		if len(ops) == consulTransactionLimit || i == len(kv.idx) {
			if ok, _, _, err := cs.client.Txn().Txn(ops, nil); !ok {
				return fmt.Errorf("execute consul SET-transaction: %w", err)
			}
			ops = nil
		}
	}

	if len(ops) > 0 {
		if ok, _, _, err := cs.client.Txn().Txn(ops, nil); !ok {
			return fmt.Errorf("execute consul SET-transaction: %w", err)
		}
	}

	return nil
}

func (cs *ConsulStorage) Delete(kv *KV) error {
	for k := range kv.idx {
		_, err := cs.client.KV().Delete(k, nil)
		if err != nil {
			return fmt.Errorf("delete %q from consul: %w", k, err)
		}
	}

	return nil
}
