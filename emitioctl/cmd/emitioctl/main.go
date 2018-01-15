package main

import (
	"context"
	"fmt"

	"github.com/supershabam/emitio/emitioctl/pb"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.TODO()
	// Set up a connection to the server.
	conn, err := grpc.Dial("0.0.0.0:8080", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewEmitioClient(conn)
	mtr, err := client.MakeTransformer(ctx, &pb.MakeTransformerRequest{
		Javascript: []byte(`function transform(acc, lines) {
	var a = JSON.parse(acc)
	var output = []
	for (var i = 0; i < lines.length; i++) {
		a.count++
		output.push(lines[i])
	}
	return [JSON.stringify(a), output]
}`),
	})
	if err != nil {
		panic(err)
	}
	stream, err := client.ReadRows(ctx, &pb.ReadRowsRequest{
		TransformerId: mtr.Id,
		Accumulator:   `{"count":0}`,
		Start:         []byte(""),
		End:           []byte(""),
	})
	if err != nil {
		panic(err)
	}
	for {
		reply, err := stream.Recv()
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", reply)
	}

}
