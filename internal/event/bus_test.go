package event

import "testing"

func TestBusSubscribeAndPublish(t *testing.T) {
	b := NewBus()
	var got []Event
	b.Subscribe(FileChanged, func(ev Event) { got = append(got, ev) })

	b.Publish(Event{ID: "1", Type: FileChanged, Source: "editor"})
	b.Publish(Event{ID: "2", Type: FileSaved, Source: "editor"})

	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("handler recebeu %v, esperava apenas o FileChanged", got)
	}
}

func TestBusSubscribeAll(t *testing.T) {
	b := NewBus()
	count := 0
	b.SubscribeAll(func(ev Event) { count++ })

	b.Publish(Event{Type: FileChanged})
	b.Publish(Event{Type: AgentStarted})
	b.Publish(Event{Type: TestFailed})

	if count != 3 {
		t.Fatalf("SubscribeAll recebeu %d, esperava 3", count)
	}
}

func TestBusUnsubscribe(t *testing.T) {
	b := NewBus()
	count := 0
	cancel := b.Subscribe(FileChanged, func(ev Event) { count++ })
	b.Publish(Event{Type: FileChanged})
	cancel()
	b.Publish(Event{Type: FileChanged})

	if count != 1 {
		t.Fatalf("após cancelar, count = %d, esperava 1", count)
	}
}

func TestBusTimestampAutofill(t *testing.T) {
	b := NewBus()
	var tsSet bool
	b.Subscribe(FileChanged, func(ev Event) { tsSet = !ev.Timestamp.IsZero() })
	b.Publish(Event{Type: FileChanged})
	if !tsSet {
		t.Fatal("Timestamp não foi preenchido")
	}
}

func TestBusSubscriberCount(t *testing.T) {
	b := NewBus()
	b.Subscribe(FileChanged, func(ev Event) {})
	b.Subscribe(FileSaved, func(ev Event) {})
	b.SubscribeAll(func(ev Event) {})
	if b.SubscriberCount() != 3 {
		t.Fatalf("SubscriberCount = %d, want 3", b.SubscriberCount())
	}
}
