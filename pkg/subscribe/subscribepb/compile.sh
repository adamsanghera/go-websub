# Remove the dead weeds, if they exist
rm -rf *.pb.go > /dev/null 2>&1

# Compile the files
protoc --go_out=plugins=grpc:$GOPATH/src/github.com/adamsanghera/go-websub/pkg/subscribe/subscribepb \
       ./service.proto