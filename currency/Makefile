.PHONY: protos

protos:
	protoc -I=protos \
		--go_out=paths=source_relative:protos \
		--go-grpc_out=paths=source_relative:protos \
		protos/currency.proto
