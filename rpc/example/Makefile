rpc:
	make -f $(GOPATH)/src/github.com/xinlaini/golibs/rpc/Makefile

protos:
	gopro rpc/example:say_proto && gopro rpc/example:sing_proto

genrpc:
	genrpc --out_dir=rpc/example $(PROTOROOT)/rpc/example/hello.svc.pb

server: protos rpc
	cd server && go build -o ./server

client: protos genrpc rpc
	cd client && go build -o ./client
