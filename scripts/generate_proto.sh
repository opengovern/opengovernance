protoc --go_out=pkg/describe/proto/src/golang --go_opt=paths=source_relative \
    --go-grpc_out=pkg/describe/proto/src/golang --go-grpc_opt=paths=source_relative \
    pkg/describe/proto/*.proto
mv pkg/describe/proto/src/golang/pkg/describe/proto/* pkg/describe/proto/src/golang/
rm -rf pkg/describe/proto/src/golang/pkg