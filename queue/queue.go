package queue;

type Queue struct {
	maxSize int64;
	values []float64;
}

func (q *Queue) Length() int64 {
	return int64(len(q.values));
}

func New(size int64) *Queue {
	q := &Queue{};
	q.maxSize = size;
	return q;
}

func (q *Queue) Enqueue(node float64) {
	q.values = append(q.values, node);
	for q.Length() > q.maxSize {
		_ = q.Dequeue();
	}
}

func (q *Queue) Dequeue() float64 {
	var val float64;
	if q.IsEmpty() {
		return 0.0;
	}
	val = q.values[0];
	q.values = q.values[1:];
	return val;
}

func (q *Queue) IsEmpty() bool {
	return q.Length() == 0;
}

func (q *Queue) Drain() {
	for !q.IsEmpty() {
		q.Dequeue();
	}
}

func (q *Queue) Inspect() []float64 {
	return q.values;
}
