rpc:
	make -f $(GOPATH)/src/github.com/xinlaini/golibs/rpc/Makefile

protos:
	gopro rpc/example:say_proto && gopro rpc/example:sing_proto

rpcgen:
	rpcgen --out_dir=rpc/example /Users/xinlaini/code/proto/rpc/example/hello.svc.pb

server: protos rpc
	cd server && go build -o ./server

client: protos rpcgen rpc
	cd client && go build -o ./client
