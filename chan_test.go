package chant

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewChannel(t *testing.T) {

	ctx := context.Background()

	c1 := New[int](ctx)
	defer c1.Close()

	c2 := New[bool](ctx)
	defer c2.Close()

	for i := 0; i < 10000; i++ {

		go func(n int) {
			i, _ := c1.Recv(ctx)
			c2.Send(ctx, i == n)
		}(i)

		c1.Send(ctx, i)
		res, err := c2.Recv(ctx)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res {
			t.Fatalf("unexpected receive on: %v", i)
		}
	}
}

func TestSendOnClosedChannel(t *testing.T) {

	ctx := context.Background()

	c := New[int](ctx)
	c.Close()

	err := c.Send(ctx, 9)
	if err == nil {
		t.Fatal("should have received error")
	}
	if !errors.Is(err, ErrChannelClosed) {
		t.Fatalf("unexpected error: got (%v), expected (%v)", err, ErrChannelClosed)
	}

}

func TestRecvOnClosedChannel(t *testing.T) {

	ctx := context.Background()

	c := New[int](ctx)
	c.Close()

	_, err := c.Recv(ctx)
	if err == nil {
		t.Fatal("should have received error")
	}
	if !errors.Is(err, ErrChannelClosed) {
		t.Fatalf("unexpected error: got (%v), expected (%v)", err, ErrChannelClosed)
	}

}

func TestRawOnClosedChannel(t *testing.T) {

	c := New[int](context.Background())
	c.Close()

	ch := c.RawChan()
	if ch != nil {
		t.Fatal("should have received nil")
	}
}

func TestBufferedChan(t *testing.T) {

	ctx := context.Background()

	c := New[int](ctx, WithSize(10))
	defer c.Close()

	exit := New[bool](ctx)
	defer exit.Close()

	exited := New[bool](ctx)
	defer exited.Close()

	total := 0

	go func() {
		for {
			select {
			case i := <-c.RawChan():
				total += i
			case <-exit.RawChan():
				{
					exited.Send(ctx, true)
					return
				}
			}
		}
	}()

	N := 1000

	for i := 0; i < N; i++ {
		c.Send(ctx, i)
	}

	time.Sleep(1 * time.Millisecond)

	exit.Send(ctx, true)
	exited.Recv(ctx)

	NTot := N * (N - 1) / 2
	if total != NTot {
		t.Fatalf("expected %v, got %v", NTot, total)
	}
}

func TestBufferedChan2(t *testing.T) {

	ctx := context.Background()

	c := New[int](ctx, WithRecvTimeout(1*time.Millisecond), WithSize(10))
	defer c.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	total := 0

	go func() {
		defer wg.Done()
		for {
			i, err := c.Recv(ctx)
			if err != nil {
				return
			}
			total += i
		}
	}()

	N := 1000

	for i := 0; i < N; i++ {
		c.Send(ctx, i)
	}

	wg.Wait()

	NTot := N * (N - 1) / 2
	if total != NTot {
		t.Fatalf("expected %v, got %v", NTot, total)
	}
}

func ExampleNew() {

	ctx := context.Background()

	c1 := New[int](ctx)
	defer c1.Close()

	c2 := New[bool](ctx)
	defer c2.Close()

	v := 42

	go func() {
		i, err := c1.Recv(ctx)
		c2.Send(ctx, err == nil && i == v)
	}()

	c1.Send(ctx, v)

	answer, _ := c2.Recv(ctx)
	fmt.Println(answer)
	// Output: true
}

func BenchmarkNew(b *testing.B) {

	ctx := context.Background()

	type msg struct {
		v  int
		ch *Channel[bool]
	}

	v := 42

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		c1 := New[*msg](ctx, WithSize(1))
		go func() {
			defer c1.Close()
			msg, err := c1.Recv(ctx)
			msg.ch.Send(ctx, err == nil && msg.v == v)
		}()

		m := &msg{
			v:  v,
			ch: New[bool](ctx),
		}
		c1.Send(ctx, m)

		answer, err := m.ch.Recv(ctx)
		m.ch.Close()
		if err != nil {
			b.Fatal(err)
		}
		if answer != true {
			b.Fatal("transmission error")
		}
	}
}
