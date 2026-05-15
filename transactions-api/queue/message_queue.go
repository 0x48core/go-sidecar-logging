package queue

// MessageQueue is a non-blocking, thread-safe FIFO backed by a buffered channel.
// Producers call Enqueue; the background writer calls DrainAll on each tick.
type MessageQueue struct {
	ch chan string
}

func NewMessageQueue(size int) *MessageQueue {
	return &MessageQueue{
		ch: make(chan string, size),
	}
}

// Enqueue adds msg to the queue. Returns false and drops the message if the
// queue is full — callers should log a warning when this happens.
func (mq *MessageQueue) Enqueue(message string) bool {
	select {
	case mq.ch <- message:
		return true
	default:
		return false
	}
}

// DrainAll returns every message currently in the queue without blocking.
// Safe to call concurrently — the channel read is atomic per message.
func (mq *MessageQueue) DrainAll() []string {
	var messages []string
	for {
		select {
		case message := <-mq.ch:
			messages = append(messages, message)
		default:
			return messages
		}
	}
}

// Len returns the current number of queued messages. Useful for metrics.
func (q *MessageQueue) Len() int {
	return len(q.ch)
}
