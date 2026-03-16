package rocketmq

import (
	"testing"
)

func TestSendReceipt_Fields(t *testing.T) {
	r := &SendReceipt{
		MessageID:     "msg-123",
		TransactionID: "tx-456",
		Offset:        100,
	}
	if r.MessageID != "msg-123" {
		t.Errorf("MessageID = %s, want msg-123", r.MessageID)
	}
	if r.TransactionID != "tx-456" {
		t.Errorf("TransactionID = %s, want tx-456", r.TransactionID)
	}
	if r.Offset != 100 {
		t.Errorf("Offset = %d, want 100", r.Offset)
	}
}

func TestMessage_Fields(t *testing.T) {
	m := &Message{
		Topic: "test-topic",
		Body:  []byte("hello"),
		Keys:  []string{"key1", "key2"},
		Tag:   "tag1",
	}
	if m.Topic != "test-topic" {
		t.Errorf("Topic = %s, want test-topic", m.Topic)
	}
	if string(m.Body) != "hello" {
		t.Errorf("Body = %s, want hello", string(m.Body))
	}
	if len(m.Keys) != 2 {
		t.Errorf("Keys length = %d, want 2", len(m.Keys))
	}
	if m.Tag != "tag1" {
		t.Errorf("Tag = %s, want tag1", m.Tag)
	}
}
