/*
If this program is on the path of your machine you can invoke it in the following way:

protoc --plugin protoc-gen-goexample --goexample_out=output example.proto

Note that the `goexample` term is both the last portion of the binary build and the first portion of the out argument.
If you named your plugin `protoc-gen-poodle` then you would need to invoke that plugin by:

protoc --plugin protoc-gen-poodle --poodle_out=output example.proto

Parameters may be set for additional information

protoc --plugin protoc-gen-goexample --goexample_out=param1=value1,param2=value2:output example.proto

I believe an equivalent, cleaner, way to do this would be using the opt argument

protoc --plugin ./protoc-gen-goexample --goexample_out=output --goexample_opt=param1=value1,param2=value2 example.proto

Parameters shall apply to multiple files.  See an example in generateCode for applying settings to individual message types using annotations.

*/
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gogo/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

type GoExample struct {
	Request    *plugin.CodeGeneratorRequest
	Response   *plugin.CodeGeneratorResponse
	Parameters map[string]string
}

type LocationMessage struct {
	Location        *descriptor.SourceCodeInfo_Location
	Message         *descriptor.DescriptorProto
	LeadingComments []string
}

func (runner *GoExample) PrintParameters(w io.Writer) {
	const padding = 3
	tw := tabwriter.NewWriter(w, 0, 0, padding, ' ', tabwriter.TabIndent)
	fmt.Fprintf(tw, "Parameters:\n")
	for k, v := range runner.Parameters {
		fmt.Fprintf(tw, "%s:\t%s\n", k, v)
	}
	fmt.Fprintln(tw, "")
	tw.Flush()
}

func (runner *GoExample) getLocationMessage() map[string][]*LocationMessage {

	ret := make(map[string][]*LocationMessage)
	for index, filename := range runner.Request.FileToGenerate {
		locationMessages := make([]*LocationMessage, 0)
		proto := runner.Request.ProtoFile[index]
		desc := proto.GetSourceCodeInfo()
		locations := desc.GetLocation()
		for _, location := range locations {
			// I would encourage developers to read the documentation about paths as I might have misunderstood this
			// I am trying to process message types which I understand to be `4` and only at the root level which I understand
			// to be path len == 2
			if len(location.GetPath()) > 2 {
				continue
			}

			leadingComments := strings.Split(location.GetLeadingComments(), "\n")
			if len(location.GetPath()) > 1 && location.GetPath()[0] == int32(4) {
				message := proto.GetMessageType()[location.GetPath()[1]]
				println(message.GetName())
				locationMessages = append(locationMessages, &LocationMessage{
					Message:  message,
					Location: location,
					// Because we are only parsing messages here at the root level we will not get field comments
					LeadingComments: leadingComments[:len(leadingComments)-1],
				})
			}
		}
		ret[filename] = locationMessages
	}
	return ret
}

func (runner *GoExample) CreateMarkdownFile(filename string, messages []*LocationMessage) error {
	// Create a file and append it to the output files

	var outfileName string
	var content string
	outfileName = strings.Replace(filename, ".proto", ".md", -1)
	var mdFile plugin.CodeGeneratorResponse_File
	mdFile.Name = &outfileName
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s\n", outfileName))
	for _, locationMessage := range messages {
		buf.WriteString(fmt.Sprintf("\n## %s\n", locationMessage.Message.GetName()))
		buf.WriteString(fmt.Sprintf("### %s\n", "Leading Comments"))
		for _, comment := range locationMessage.LeadingComments {
			buf.WriteString(fmt.Sprintf("%s\n", comment))
		}
		if len(locationMessage.Message.NestedType) > 0 {
			buf.WriteString(fmt.Sprintf("### %s\n", "Nested Messages"))
			for _, nestedMessage := range locationMessage.Message.NestedType {
				buf.WriteString(fmt.Sprintf("#### %s\n", nestedMessage.GetName()))
				buf.WriteString(fmt.Sprintf("#### %s\n", "Fields"))
				for _, field := range nestedMessage.Field {
					buf.WriteString(fmt.Sprintf("%s - %s\n", field.GetName(), field.GetLabel()))
				}
			}
		}
		for _, field := range locationMessage.Message.Field {
			buf.WriteString(fmt.Sprintf("%s - %s\n", field.GetName(), field.GetLabel()))
		}
	}
	content = buf.String()
	mdFile.Content = &content
	runner.Response.File = append(runner.Response.File, &mdFile)
	os.Stderr.WriteString(fmt.Sprintf("Created File: %s", filename))
	return nil
}

func (runner *GoExample) generateMessageMarkdown() error {
	// This convenience method will return a structure of some types that I use
	fileLocationMessageMap := runner.getLocationMessage()
	for filename, locationMessages := range fileLocationMessageMap {
		runner.CreateMarkdownFile(filename, locationMessages)
	}
	return nil
}

func (runner *GoExample) generateCode() error {
	// Initialize the output file slice
	files := make([]*plugin.CodeGeneratorResponse_File, 0)
	runner.Response.File = files

	{
		err := runner.generateMessageMarkdown()
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	// os.Stdin will contain data which will unmarshal into the following object:
	// https://godoc.org/github.com/golang/protobuf/protoc-gen-go/plugin#CodeGeneratorRequest
	req := &plugin.CodeGeneratorRequest{}
	resp := &plugin.CodeGeneratorResponse{}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	// You must use the requests unmarshal method to handle this type
	err = req.Unmarshal(data)
	if err != nil {
		panic(err)
	}

	// You may require more data than what is in the proto files alone.  There are a couple ways in which to do this.
	// The first is by parameters.  Another may be using leading comments in the proto files which I will cover in generateCode.
	parameters := req.GetParameter()
	// =grpc,import_path=mypackage:.
	exampleRunner := &GoExample{
		Request:    req,
		Response:   resp,
		Parameters: make(map[string]string),
	}
	groupkv := strings.Split(parameters, ",")
	for _, element := range groupkv {
		kv := strings.Split(element, "=")
		if len(kv) > 1 {
			exampleRunner.Parameters[kv[0]] = kv[1]
		}
	}
	// Print the parameters for example
	exampleRunner.PrintParameters(os.Stderr)

	err = exampleRunner.generateCode()
	if err != nil {
		panic(err)
	}

	marshalled, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(marshalled)
}
