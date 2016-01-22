rpc:
	make -f ../../Makefile precondition

proto:
	gopro rpc/tools/rpcc:def_proto

build: rpc proto
	go build -o ./rpcc
