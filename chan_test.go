package chant

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewChannel(t *testing.T) {

	c1 := New[int]()
	defer c1.Close()

	c2 := New[bool]()
	defer c2.Close()

	for i := 0; i < 10000; i++ {

		go func(n int) {
			i, _ := c1.Recv()
			c2.Send(i == n)
		}(i)

		c1.Send(i)
		res, err := c2.Recv()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res {
			t.Fatalf("unexpected receive on: %v", i)
		}
	}
}

func TestSendOnClosedChannel(t *testing.T) {

	c := New[int]()
	c.Close()

	err := c.Send(9)
	if err == nil {
		t.Fatal("should have received error")
	}
	if !errors.Is(err, ErrChannelClosed) {
		t.Fatalf("unexpected error: got (%v), expected (%v)", err, ErrChannelClosed)
	}

}

func TestRecvOnClosedChannel(t *testing.T) {

	c := New[int]()
	c.Close()

	_, err := c.Recv()
	if err == nil {
		t.Fatal("should have received error")
	}
	if !errors.Is(err, ErrChannelClosed) {
		t.Fatalf("unexpected error: got (%v), expected (%v)", err, ErrChannelClosed)
	}

}

func TestRawOnClosedChannel(t *testing.T) {

	c := New[int]()
	c.Close()

	ch := c.RawChan()
	if ch != nil {
		t.Fatal("should have received nil")
	}
}

func TestBufferedChan(t *testing.T) {

	c := New[int](WithSize(10))
	defer c.Close()

	exit := New[bool]()
	defer exit.Close()

	exited := New[bool]()
	defer exited.Close()

	total := 0

	go func() {
		for {
			select {
			case i := <-c.RawChan():
				total += i
			case <-exit.RawChan():
				{
					exited.Send(true)
					return
				}
			}
		}
	}()

	N := 1000

	for i := 0; i < N; i++ {
		c.Send(i)
	}

	time.Sleep(1 * time.Millisecond)

	exit.Send(true)
	exited.Recv()

	NTot := N * (N - 1) / 2
	if total != NTot {
		t.Fatalf("expected %v, got %v", NTot, total)
	}
}

func TestBufferedChan2(t *testing.T) {

	c := New[int](WithTimeout(1*time.Millisecond), WithSize(10))
	defer c.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	total := 0

	go func() {
		defer wg.Done()
		for {
			i, err := c.Recv()
			if err != nil {
				return
			}
			total += i
		}
	}()

	N := 1000

	for i := 0; i < N; i++ {
		c.Send(i)
	}

	wg.Wait()

	NTot := N * (N - 1) / 2
	if total != NTot {
		t.Fatalf("expected %v, got %v", NTot, total)
	}
}

func ExampleNew() {

	c1 := New[int]()
	defer c1.Close()

	c2 := New[bool]()
	defer c2.Close()

	v := 42

	go func() {
		i, err := c1.Recv()
		c2.Send(err == nil && i == v)
	}()

	c1.Send(v)

	answer, _ := c2.Recv()
	fmt.Println(answer)
	// Output: true
}

func BenchmarkNew(b *testing.B) {

	type msg struct {
		v  int
		ch *Channel[bool]
	}

	v := 42

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		c1 := New[*msg](WithSize(1))
		go func() {
			defer c1.Close()
			msg, err := c1.Recv()
			msg.ch.Send(err == nil && msg.v == v)
		}()

		m := &msg{
			v:  v,
			ch: New[bool](),
		}
		c1.Send(m)

		answer, err := m.ch.Recv()
		m.ch.Close()
		if err != nil {
			b.Fatal(err)
		}
		if answer != true {
			b.Fatal("transmission error")
		}
	}
}
