package queue;

type Queue struct {
	maxSize int;
	values []float64;
}

func New(size int) *Queue {
	queue := &Queue{};
	queue.maxSize = size;
	return queue;
}

func (q *Queue) Enqueue(node float64) {
	q.values = append(q.values, node);
	for len(q.values) > q.maxSize {
		_ = q.Dequeue();
	}
}

func (q *Queue) Dequeue() float64 {
	var val float64
	if q.IsEmpty() {
		return 0.0
	}
	val = q.values[0]
	q.values = q.values[1:]
	return val
}

func (q *Queue) IsEmpty() bool {
	return len(q.values) == 0
}

func (q *Queue) Inspect() []float64 {
	return q.values;
}
