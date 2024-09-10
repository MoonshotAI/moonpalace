package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"hash"
	"hash/fnv"
	"sync"

	"github.com/tidwall/pretty"
	defc "github.com/x5iu/defc/runtime"
)

var caching = NewCaching()

//go:generate defc generate --func endpoint=getEndpoint
type Caching interface {
	Response() *defc.JSON

	// Create POST Scan(cache) {{ endpoint }}/v1/caching
	// Content-Type: application/json
	// Authorization: Bearer {{ .key }}
	Create(ctx context.Context, key string, cache *Cache) error

	// Get GET {{ endpoint }}/v1/caching/{{ .id }}
	// Authorization: Bearer {{ .key }}
	Get(ctx context.Context, key string, id string) (*Cache, error)

	// Delete DELETE {{ endpoint }}/v1/caching/{{ .id }}
	// Authorization: Bearer {{ .key }}
	Delete(ctx context.Context, key string, id string) error
}

type Cache struct {
	defc.JSONBody[Cache]

	Messages json.RawMessage `json:"messages"`
	Tools    json.RawMessage `json:"tools"`
	TTL      int             `json:"ttl"`

	ID        string `json:"id"`
	Status    string `json:"status"`
	ExpiredAt int    `json:"expired_at"`
}

func (c *Cache) MarshalJSON() ([]byte, error) {
	type Marshaler struct {
		Model    string          `json:"model"`
		Messages json.RawMessage `json:"messages"`
		Tools    json.RawMessage `json:"tools"`
		TTL      int             `json:"ttl"`
	}
	return json.Marshal(&Marshaler{
		Model:    "moonshot-v1",
		Messages: c.Messages,
		Tools:    c.Tools,
		TTL:      c.TTL,
	})
}

func getEndpoint() string {
	return endpoint
}

var (
	hasherPool = &sync.Pool{
		New: func() any {
			return fnv.New128()
		},
	}
	hashListPool = &sync.Pool{
		New: func() any {
			return make([]string, 0, 4)
		},
	}
)

func putHasher(hasher hash.Hash) {
	hasher.Reset()
	hasherPool.Put(hasher)
}

func putHashList(hashList []string) {
	hashListPool.Put(hashList[:0])
}

func hashKey(key string) string {
	hasher := hasherPool.Get().(hash.Hash)
	defer putHasher(hasher)
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func hashPrefix(
	cacheMinBytes int,
	tools json.RawMessage,
	messages []*MoonshotMessage,
) ([]string, int) {
	hasher := hasherPool.Get().(hash.Hash)
	defer putHasher(hasher)
	hashList := hashListPool.Get().([]string)
	defer putHashList(hashList)
	nBytes := 0
	uglyTools := pretty.Ugly(tools)
	hasher.Write(uglyTools)
	nBytes += len(uglyTools)
	for _, message := range messages {
		hasher.Write([]byte(message.Role))
		nBytes += len([]byte(message.Role))
		hasher.Write([]byte(message.Content))
		nBytes += len([]byte(message.Content))
		for _, toolCall := range message.ToolCalls {
			hasher.Write([]byte(toolCall.Function.Name))
			nBytes += len([]byte(toolCall.Function.Name))
			hasher.Write([]byte(toolCall.Function.Arguments))
			nBytes += len([]byte(toolCall.Function.Arguments))
		}
		if nBytes > cacheMinBytes {
			hashHexCode := hex.EncodeToString(hasher.Sum(nil))
			hashList = append(hashList, hashHexCode)
		}
	}
	return hashList, nBytes
}
