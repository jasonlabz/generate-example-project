package user

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type managerImpl struct {
	mu      sync.RWMutex
	records map[int64]Record
	nextID  int64
}

var _ Manager = (*managerImpl)(nil)

// NewManager 创建基于内存存储的用户 Manager。
//
// 真实项目把这里的 map 替换为 DAO / Redis / 外部 API 客户端即可，Manager 接口与上层
// 代码无需改动——这正是把技术能力收敛到 manager 层的收益。
// 这里预置两条数据，便于服务启动后直接在 Knife4j 上调通列表与详情接口。
func NewManager() Manager {
	impl := &managerImpl{records: make(map[int64]Record), nextID: 1}
	for _, name := range []string{"alice", "bob"} {
		impl.insert(Record{Name: name, Email: name + "@example.com"})
	}
	return impl
}

// insert 是无锁的内部写入，调用方需自行持锁；仅构造期与 Insert 使用。
func (m *managerImpl) insert(record Record) Record {
	record.ID = m.nextID
	m.nextID++
	m.records[record.ID] = record
	return record
}

// FindByID 按主键查询用户。
func (m *managerImpl) FindByID(_ context.Context, id int64) (Record, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, ok := m.records[id]
	return record, ok, nil
}

// FindByName 按用户名查询用户。
func (m *managerImpl) FindByName(_ context.Context, name string) (Record, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, record := range m.records {
		if record.Name == name {
			return record, true, nil
		}
	}
	return Record{}, false, nil
}

// Search 按关键字分页查询，返回当页数据与匹配总数。
func (m *managerImpl) Search(_ context.Context, keyword string, offset, limit int64) ([]Record, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// map 遍历顺序不稳定，先按主键排序保证分页结果可预期。
	all := make([]Record, 0, len(m.records))
	for _, record := range m.records {
		all = append(all, record)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	matched := all
	if keyword != "" {
		matched = make([]Record, 0, len(all))
		for _, record := range all {
			if strings.Contains(record.Name, keyword) || strings.Contains(record.Email, keyword) {
				matched = append(matched, record)
			}
		}
	}

	total := int64(len(matched))
	if limit <= 0 {
		return matched[minInt64(offset, total):], total, nil
	}
	if offset > total {
		offset = total
	}
	end := minInt64(offset+limit, total)
	return matched[offset:end], total, nil
}

// Insert 写入用户并分配主键。
func (m *managerImpl) Insert(_ context.Context, record Record) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.insert(record), nil
}

// Delete 删除用户，返回是否真的删到了记录。
func (m *managerImpl) Delete(_ context.Context, id int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.records[id]; !ok {
		return false, nil
	}
	delete(m.records, id)
	return true, nil
}

// minInt64 返回两者较小值，用于把越界的 offset/end 收敛到合法范围。
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
