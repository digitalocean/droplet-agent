package file

import "github.com/digitalocean/droplet-agent/internal/log"

// ringBuffer is a circular buffer for efficiently storing the last N lines
type ringBuffer struct {
	buffer []string
	size   int
	start  int
	count  int
}

// newRingBuffer creates a new ring buffer with the specified capacity
func newRingBuffer(capacity int) *ringBuffer {
	log.Debug("[Troubleshooting Actioner] Creating ring buffer with capacity: %d", capacity)
	return &ringBuffer{
		buffer: make([]string, capacity),
		size:   capacity,
		start:  0,
		count:  0,
	}
}

// add inserts a line into the ring buffer, overwriting the oldest line if full
func (rb *ringBuffer) add(line string) {
	if rb.size == 0 {
		return
	}

	index := (rb.start + rb.count) % rb.size
	rb.buffer[index] = line

	if rb.count < rb.size {
		rb.count++
	} else {
		rb.start = (rb.start + 1) % rb.size
	}
}

// getLines returns all lines in the ring buffer in the correct order
func (rb *ringBuffer) getLines() []string {
	if rb.count == 0 {
		return nil
	}

	result := make([]string, rb.count)
	for i := 0; i < rb.count; i++ {
		index := (rb.start + i) % rb.size
		result[i] = rb.buffer[index]
	}
	return result
}
