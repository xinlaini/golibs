package main

import (
	"flag"
	"time"

	hello_proto "gen/pb/rpc/example/hello_proto"
	"gen/pb/rpc/example/say_proto"
	"gen/pb/rpc/example/sing_proto"
	"gen/rpc/rpc/example"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
	"github.com/xinlaini/golibs/rpc"
)

func runSay(logger xlog.Logger, client *hello.HelloClient, req *say.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.Say(&rpc.ClientContext{Context: ctx}, req)
	if err != nil {
		logger.Errorf("Say error: %s", err)
	} else if resp == nil {
		logger.Info("Say returned nil resp")
	} else {
		logger.Infof("Say returned:\n%s", resp.String())
	}
}

func runSing(logger xlog.Logger, client *hello.HelloClient, req *sing.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.Sing(&rpc.ClientContext{Context: ctx}, req)
	if err != nil {
		logger.Errorf("Sing error: %s", err)
	} else if resp == nil {
		logger.Info("Sing returned nil resp")
	} else {
		logger.Infof("Sing returned:\n%s", resp.String())
	}
}

func main() {
	flag.Parse()

	logger := xlog.NewConsoleLogger()

	ctrl, err := rpc.NewController(rpc.Config{
		Logger: logger,
	})

	helloClient, err := hello.NewHelloClient(ctrl, rpc.ClientOptions{
		ServiceName:  "Hello",
		ServiceAddr:  "localhost:9090",
		ConnPoolSize: 5,
		Retry:        rpc.DefaultDialRetry,
	})
	if err != nil {
		logger.Fatalf("Failed to create client: %s", err)
	}

	runSay(logger, helloClient, nil)
	runSay(logger, helloClient, &say.Request{})
	runSay(logger, helloClient, &say.Request{Hdr: &hello_proto.Header{}, Body: proto.String("say body")})

	runSing(logger, helloClient, nil)
	runSing(logger, helloClient, &sing.Request{})
	runSing(logger, helloClient, &sing.Request{Hdr: &hello_proto.Header{}, Body: proto.String("sing body")})
	helloClient.Close()
}
