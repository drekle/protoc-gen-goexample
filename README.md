# protoc-gen-goexample
An example of a protoc plugin in go

## Build
```
go build .
```

## Run the plugin
```
protoc --plugin protoc-gen-goexample --goexample_out=output example.proto
```

## Passing additional data to generators
Parameters may be set for additional information
```
protoc --plugin protoc-gen-goexample --goexample_out=param1=value1,param2=value2:output example.proto
```
I believe an equivalent, cleaner, way to do this would be using the opt argument
```
protoc --plugin ./protoc-gen-goexample --goexample_out=output --goexample_opt=param1=value1,param2=value2 example.proto
```
Parameters shall apply to multiple files.  See an example in generateCode for parsing comments.  You might consider using annotations as comments to apply additional data at the message level, or field level, by adding annotations as leading comments.