echo "Generating code from protobuf files..." && echo

count=0
total=(`find . -name *pb -type d`)
totalCount=${#total[@]}

# This is a super ugly way to get all directories matching *pb in absolute form
for dname in `find . -name *pb -type d | cut -c 2- | while read line; do echo "\`pwd\`$line"; done`; do
  (( count++ ))

  # Remove old files, if they exist
  rm -rf $dname/*.pb.go > /dev/null 2>&1

  # Generate new files
  protoc --proto_path=$dname \
         --go_out=plugins=grpc:$dname \
         $dname/*.proto

  # Print some cool graphics
  echo "Generating files for directory $count / $totalCount"
  tree --noreport -I *.proto $dname
done