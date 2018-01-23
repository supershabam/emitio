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
	addr, _, err := server(ctx,
		WithID("id-1234"),
		WithIngresses([]Ingresser{&mockIngresser{}}),
		WithKey("key-9876"),
		WithOrigin(map[string]string{
			"dc": "nyc2",
		}),
	)
	require.Nil(t, err)
	cc, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	require.Nil(t, err)
	c := emitio.NewEmitioClient(cc)
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

func TestGetOne(t *testing.T) {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	addr, _, err := server(ctx,
		WithIngresses([]Ingresser{&mockIngresser{}}),
	)
	require.Nil(t, err)
	cc, err := grpc.Dial(addr, grpc.WithInsecure())
	require.Nil(t, err)
	c := emitio.NewEmitioClient(cc)
	// transformer that squashes all output but we still expect to get a response on each input
	mtr, err := c.MakeTransformer(ctx, &emitio.MakeTransformerRequest{
		Javascript: []byte(`
function transform(acc, line) {
	return [acc, []]
}`),
	})
	require.Nil(t, err)
	stream, err := c.ReadRows(ctx, &emitio.ReadRowsRequest{
		Start:         []byte{},
		TransformerId: mtr.Id,
		InputLimit:    2,
	})
	require.Nil(t, err)
	reply, err := stream.Recv()
	require.Nil(t, err)
	assert.Equal(t, &emitio.ReadRowsReply{
		Rows:            nil,
		LastAccumulator: "",
		LastInputRowKey: []byte("mock:///:0000000000000001"),
	}, reply)
	reply, err = stream.Recv()
	require.Nil(t, err)
	assert.Equal(t, &emitio.ReadRowsReply{
		Rows:            nil,
		LastAccumulator: "",
		LastInputRowKey: []byte("mock:///:0000000000000002"),
	}, reply)
}

type mockIngresser struct{}

func (mi mockIngresser) Ingress(ctx context.Context) (<-chan string, Wait) {
	ch := make(chan string)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		ch <- "one"
		ch <- "two"
		return nil
	})
	return ch, eg.Wait
}

// URI uniquely identifies an ingress and all its configuration.
func (mi mockIngresser) URI() string {
	return "mock:///"
}

func server(ctx context.Context, opts ...ServerOption) (string, Wait, error) {
	eg, ctx := errgroup.WithContext(ctx)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	addr := l.Addr().String()
	path, err := ioutil.TempDir("", "emitio")
	if err != nil {
		return "", nil, err
	}
	db, err := ParseStorage(fmt.Sprintf("file://%s", path))
	if err != nil {
		return "", nil, err
	}
	opts = append([]ServerOption{
		WithDB(db),
	}, opts...)
	s, err := NewServer(ctx, opts...)
	if err != nil {
		return "", nil, err
	}
	eg.Go(func() error {
		return s.Run(ctx)
	})
	eg.Go(func() error {
		grpcServer := grpc.NewServer()
		emitio.RegisterEmitioServer(grpcServer, s)
		return grpcServer.Serve(l)
	})
	return addr, eg.Wait, nil
}
