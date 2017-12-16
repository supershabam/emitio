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
	conn, err := grpc.Dial("0.0.0.0:9090", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewEmitioClient(conn)
	reply, err := client.ReadRows(ctx, &pb.ReadRowsRequest{
		Start: []byte(""),
		End:   []byte(""),
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", reply)
}
