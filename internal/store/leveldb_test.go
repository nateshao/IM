package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/user/im/internal/model"
)

func TestLevelDBStore_Basic(t *testing.T) {
	dbPath := "./testdata/leveldb"
	_ = os.RemoveAll(dbPath)
	store, err := NewLevelDBStore(dbPath)
	assert.NoError(t, err)
	defer func() {
		store.Close()
		_ = os.RemoveAll(dbPath)
	}()

	msg := &model.Message{
		ID:         "msg1",
		SenderID:   "userA",
		ReceiverID: "userB",
		Content:    "hello",
		Timestamp:  time.Now().Unix(),
		Status:     "sent",
	}

	err = store.SaveMessage(msg)
	assert.NoError(t, err)

	got, err := store.GetMessage("msg1")
	assert.NoError(t, err)
	assert.Equal(t, msg.Content, got.Content)
}

func TestLevelDBStore_OfflineMessages(t *testing.T) {
	dbPath := "./testdata/leveldb2"
	_ = os.RemoveAll(dbPath)
	store, err := NewLevelDBStore(dbPath)
	assert.NoError(t, err)
	defer func() {
		store.Close()
		_ = os.RemoveAll(dbPath)
	}()

	userID := "userB"
	msgs := []*model.Message{
		{ID: "m1", SenderID: "A", ReceiverID: userID, Content: "1", Timestamp: 1, Status: "sent"},
		{ID: "m2", SenderID: "A", ReceiverID: userID, Content: "2", Timestamp: 2, Status: "sent"},
		{ID: "m3", SenderID: "A", ReceiverID: userID, Content: "3", Timestamp: 3, Status: "sent"},
	}
	for _, m := range msgs {
		err := store.SetOfflineMessage(userID, m)
		assert.NoError(t, err)
	}

	got, err := store.GetOfflineMessages(userID, "", 10)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, "1", got[0].Content)

	// 删除一条
	err = store.RemoveOfflineMessage(userID, "m1")
	assert.NoError(t, err)
	got, err = store.GetOfflineMessages(userID, "", 10)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(got))
}

func TestLevelDBStore_Concurrent(t *testing.T) {
	dbPath := "./testdata/leveldb3"
	_ = os.RemoveAll(dbPath)
	store, err := NewLevelDBStore(dbPath)
	assert.NoError(t, err)
	defer func() {
		store.Close()
		_ = os.RemoveAll(dbPath)
	}()

	userID := "userC"
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			msg := &model.Message{
				ID:         "mc" + string(i),
				SenderID:   "A",
				ReceiverID: userID,
				Content:    "c",
				Timestamp:  int64(i),
				Status:     "sent",
			}
			_ = store.SetOfflineMessage(userID, msg)
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = store.GetOfflineMessages(userID, "", 100)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}
