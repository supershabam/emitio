package pkg

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supershabam/emitio/emitio/pb/emitio"
	"github.com/supershabam/emitio/emitio/pkg/transformers"
	"google.golang.org/grpc"
)

func TestTransform(t *testing.T) {
	tr, err := transformers.NewJS(`
	function transform(acc, line) {
		var a = JSON.parse(acc)
		var output = []
		a.count++
		output.push(line)
		output.push(line)
		return [JSON.stringify(a), output]
	}
`)
	require.Nil(t, err)
	ctx := context.TODO()
	acc := `{"count":0}`
	in := []string{
		"{\"a\":\"2018-01-15T12:07:24.186726127-08:00\",\"r\":\"sldfkjsdjklfhi\\n\",\"s\":1}",
		"{\"a\":\"2018-01-15T12:12:32.977232909-08:00\",\"r\":\"sldfkjsdjklfhi\\n\",\"s\":2}",
	}
	acc, out, err := tr.Transform(ctx, acc, in[0])
	require.Nil(t, err)
	assert.Equal(t, `{"count":1}`, acc)
	assert.Equal(t, 2, len(out))
	acc, out, err = tr.Transform(ctx, acc, in[1])
	assert.Nil(t, err)
}

func TestInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := server(ctx,
		WithID("id-1234"),
		WithIngresses([]Ingresser{&mockIngresser{}}),
		WithKey("key-9876"),
		WithOrigin(map[string]string{
			"dc": "nyc2",
		}),
	)
	require.Nil(t, err)
	reply, err := c.Info(ctx, &emitio.InfoRequest{})
	require.Nil(t, err)
	assert.Equal(t, &emitio.InfoReply{
		Key: "key-9876",
		Id:  "id-1234",
		Origin: map[string]string{
			"dc": "nyc2",
		},
		Ingresses: []string{"mock:///"},
	}, reply)
}

func TestBatch(t *testing.T) {
	t.Run("empty database", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		path, err := ioutil.TempDir("", "emitio")
		require.Nil(t, err)
		db, err := ParseStorage(fmt.Sprintf("file:///%s", path))
		require.Nil(t, err)
		s, err := NewServer(ctx, WithDB(db))
		require.Nil(t, err)
		rowCh, wait := s.batch(ctx, []byte{}, []byte{}, 1)
		rows, active := <-rowCh
		require.Nil(t, rows)
		require.False(t, active)
		err = wait()
		require.Nil(t, err)
	})
	t.Run("one at a time with db of two", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		path, err := ioutil.TempDir("", "emitio")
		require.Nil(t, err)
		db, err := ParseStorage(fmt.Sprintf("file:///%s", path))
		require.Nil(t, err)
		err = db.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte("b"), []byte("second"))
			if err != nil {
				return err
			}
			err = txn.Set([]byte("a"), []byte("first"))
			if err != nil {
				return err
			}
			return nil
		})
		require.Nil(t, err)
		s, err := NewServer(ctx, WithDB(db))
		require.Nil(t, err)
		rowCh, wait := s.batch(ctx, []byte{}, []byte{}, 1)
		rows, active := <-rowCh
		require.True(t, active)
		require.Equal(t, rows, []row{{key: []byte("a"), value: "first"}})
		rows, active = <-rowCh
		require.True(t, active)
		require.Equal(t, rows, []row{{key: []byte("b"), value: "second"}})
		rows, active = <-rowCh
		require.False(t, active)
		require.Nil(t, rows)
		err = wait()
		require.Nil(t, err)
	})
	t.Run("three at a time with db of two", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		path, err := ioutil.TempDir("", "emitio")
		require.Nil(t, err)
		db, err := ParseStorage(fmt.Sprintf("file:///%s", path))
		require.Nil(t, err)
		err = db.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte("b"), []byte("second"))
			if err != nil {
				return err
			}
			err = txn.Set([]byte("a"), []byte("first"))
			if err != nil {
				return err
			}
			return nil
		})
		require.Nil(t, err)
		s, err := NewServer(ctx, WithDB(db))
		require.Nil(t, err)
		rowCh, wait := s.batch(ctx, []byte{}, []byte{}, 3)
		rows, active := <-rowCh
		require.True(t, active)
		require.Equal(t, rows, []row{
			{key: []byte("a"), value: "first"},
			{key: []byte("b"), value: "second"},
		})
		rows, active = <-rowCh
		require.False(t, active)
		require.Nil(t, rows)
		err = wait()
		require.Nil(t, err)
	})
	t.Run("three at a time with db of two ending after first entry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		path, err := ioutil.TempDir("", "emitio")
		require.Nil(t, err)
		db, err := ParseStorage(fmt.Sprintf("file:///%s", path))
		require.Nil(t, err)
		err = db.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte("b"), []byte("second"))
			if err != nil {
				return err
			}
			err = txn.Set([]byte("a"), []byte("first"))
			if err != nil {
				return err
			}
			return nil
		})
		require.Nil(t, err)
		s, err := NewServer(ctx, WithDB(db))
		require.Nil(t, err)
		rowCh, wait := s.batch(ctx, []byte{}, []byte("b"), 3)
		rows, active := <-rowCh
		require.True(t, active)
		require.Equal(t, rows, []row{{key: []byte("a"), value: "first"}})
		rows, active = <-rowCh
		require.False(t, active)
		require.Nil(t, rows)
		err = wait()
		require.Nil(t, err)
	})
	t.Run("three at a time with db of two starting after first entry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		path, err := ioutil.TempDir("", "emitio")
		require.Nil(t, err)
		db, err := ParseStorage(fmt.Sprintf("file:///%s", path))
		require.Nil(t, err)
		err = db.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte("b"), []byte("second"))
			if err != nil {
				return err
			}
			err = txn.Set([]byte("a"), []byte("first"))
			if err != nil {
				return err
			}
			return nil
		})
		require.Nil(t, err)
		s, err := NewServer(ctx, WithDB(db))
		require.Nil(t, err)
		rowCh, wait := s.batch(ctx, []byte("b"), []byte(""), 3)
		rows, active := <-rowCh
		require.True(t, active)
		require.Equal(t, rows, []row{{key: []byte("b"), value: "second"}})
		rows, active = <-rowCh
		require.False(t, active)
		require.Nil(t, rows)
		err = wait()
		require.Nil(t, err)
	})
}

func TestGetOne(t *testing.T) {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, _, err := server(ctx,
		WithIngresses([]Ingresser{&mockIngresser{msgs: []string{"one"}}}),
	)
	require.Nil(t, err)
	// transformer that squashes all output but we still expect to get a response on each input
	mtr, err := c.MakeTransformer(ctx, &emitio.MakeTransformerRequest{
		Javascript: []byte(`
function transform(acc, line) {
	return [acc, []]
}`),
	})
	require.Nil(t, err)
	time.Sleep(time.Second)
	stream, err := c.Read(ctx, &emitio.ReadRequest{
		Start:         []byte{},
		TransformerId: mtr.Id,
		InputLimit:    1,
	})
	require.Nil(t, err)
	reply, err := stream.Recv()
	require.Nil(t, err)
	assert.Equal(t, &emitio.ReadReply{
		Rows:            nil,
		LastAccumulator: "",
		LastInputKey:    []byte("mock:///:0000000000000001"),
	}, reply)
	reply, err = stream.Recv()
	require.Nil(t, err)
	assert.Equal(t, &emitio.ReadReply{
		Rows:            nil,
		LastAccumulator: "",
		LastInputKey:    []byte("mock:///:0000000000000002"),
	}, reply)
}

type mockIngresser struct {
	name string
	msgs []string
}

func (mi *mockIngresser) Ingress(ctx context.Context) (<-chan string, Wait) {
	ch := make(chan string)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		if mi.msgs == nil {
			return nil
		}
		for _, msg := range mi.msgs {
			select {
			case <-ctx.Done():
				return nil
			case ch <- msg:
			}
		}
		return nil
	})
	return ch, eg.Wait
}

// URI uniquely identifies an ingress and all its configuration.
func (mi mockIngresser) URI() string {
	return fmt.Sprintf("mock:///%s", mi.name)
}

func server(ctx context.Context, opts ...ServerOption) (emitio.EmitioClient, Wait, error) {
	eg, ctx := errgroup.WithContext(ctx)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	addr := l.Addr().String()
	path, err := ioutil.TempDir("", "emitio")
	if err != nil {
		return nil, nil, err
	}
	db, err := ParseStorage(fmt.Sprintf("file://%s", path))
	if err != nil {
		return nil, nil, err
	}
	opts = append([]ServerOption{
		WithDB(db),
	}, opts...)
	s, err := NewServer(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}
	eg.Go(func() error {
		return s.Run(ctx)
	})
	eg.Go(func() error {
		grpcServer := grpc.NewServer()
		emitio.RegisterEmitioServer(grpcServer, s)
		return grpcServer.Serve(l)
	})
	cc, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	c := emitio.NewEmitioClient(cc)
	return c, eg.Wait, nil
}
