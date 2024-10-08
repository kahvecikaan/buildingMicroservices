.PHONY: protos

protos:
	protoc -I protos/ protos/currency.proto \
	--go_out=. \
	--go_opt=module=github.com/kahvecikaan/buildingMicroservices/currency \
	--go-grpc_out=. \
	--go-grpc_opt=module=github.com/kahvecikaan/buildingMicroservices/currency