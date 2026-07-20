package bucketmigration

import (
	"sort"
	"sync"
)

var (
	migrations []Migration
	mu         sync.Mutex
)

func Register(m Migration) {
	mu.Lock()
	migrations = append(migrations, m)
	mu.Unlock()
}

func ClearRegisteredMigrations() {
	mu.Lock()
	migrations = []Migration{}
	mu.Unlock()
}

func All() []Migration {
	mu.Lock()
	sort.SliceStable(migrations, func(i, j int) bool {
		return migrations[i].Timestamp().Before(migrations[j].Timestamp())
	})
	defer mu.Unlock()
	return migrations
}
