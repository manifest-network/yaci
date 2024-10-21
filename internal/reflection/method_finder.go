package reflection

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// FindMethodDescriptor finds the method descriptor for a given service and method name.
func FindMethodDescriptor(files *protoregistry.Files, serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	var methodDesc protoreflect.MethodDescriptor
	var found bool
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			if string(svc.FullName()) == serviceName {
				methods := svc.Methods()
				for j := 0; j < methods.Len(); j++ {
					m := methods.Get(j)
					if string(m.Name()) == methodName {
						methodDesc = m
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	if !found {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
	}
	return methodDesc, nil
}
