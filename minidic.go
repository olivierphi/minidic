package minidic

import (
	"fmt"
	"reflect"
)

type container struct {
	injections            map[string]injection
	functionsResultsCache map[string]*interface{}
}

type injection struct {
	injectionId string
	value       interface{}
	asFactory   bool
	protected   bool
}

func NewContainer() *container {
	c := new(container)
	c.injections = make(map[string]injection)
	c.functionsResultsCache = make(map[string]*interface{})
	return c
}

func NewInjection(injectionId string, value interface{}) injection {
	return injection{injectionId: injectionId, value: value, asFactory: false, protected: false}
}

func (c *container) add(injection injection) {
	c.injections[injection.injectionId] = injection
}

func (c *container) get(injectionId string) interface{} {
	value, err := c.getWithoutPanic(injectionId)
	if err != nil {
		panic(err)
	}
	return value
}

func (c *container) getWithoutPanic(injectionId string) (interface{}, error) {
	if value, ok := c.functionsResultsCache[injectionId]; ok {
		return *value, nil
	}

	injection, exists := c.injections[injectionId]
	if !exists {
		return nil, UnknownInjectionIdError{injectionId: injectionId}
	}

	value := injection.value

	if isFunction(value) && !injection.protected {
		result, err := triggerFunctionWithContainer(injectionId, c, value)
		if err != nil {
			return nil, err
		}
		if !injection.asFactory {
			c.functionsResultsCache[injectionId] = &result
		}

		return result, nil
	}

	return value, nil
}

func (c *container) has(injectionId string) bool {
	_, ok := c.injections[injectionId]
	return ok
}

func (c *container) del(injectionId string) error {
	_, ok := c.injections[injectionId]
	if !ok {
		return UnknownInjectionIdError{injectionId: injectionId}
	}

	delete(c.injections, injectionId)

	return nil
}

func (c *container) extend(injectionId string, function interface{}) {
	extendedInjection, ok := c.injections[injectionId]
	if !ok {
		panic(UnknownInjectionIdError{injectionId: injectionId})
	}
	extendedInjectionFunction := extendedInjection.value
	if !isFunction(extendedInjectionFunction) {
		panic(ExtendedServiceIsNotAFunctionError{extendedInjectionId: injectionId})
	}
	if !isFunction(function) {
		panic(ServiceExtensionIsNotAFunctionError{extendedInjectionId: injectionId})
	}

	extendedInjection.value = func(c *container) interface{} {
		decoratedInjectionResult, err := triggerFunctionWithContainer(injectionId, c, extendedInjectionFunction)
		if err != nil {
			panic(err)
		}
		result, err := triggerFunctionWithContainer(injectionId, c, function, decoratedInjectionResult)
		if err != nil {
			panic(err)
		}
		return result
	}
	c.injections[injectionId] = extendedInjection
}

type UnknownInjectionIdError struct {
	injectionId string
}

func (e UnknownInjectionIdError) Error() string {
	return fmt.Sprintf("Unknown injection id '%s'", e.injectionId)
}

type ExtendedServiceIsNotAFunctionError struct {
	extendedInjectionId string
}

func (e ExtendedServiceIsNotAFunctionError) Error() string {
	return fmt.Sprintf("Extended injection id '%s' is not mapped to a function", e.extendedInjectionId)
}

type ServiceExtensionIsNotAFunctionError struct {
	extendedInjectionId string
}

func (e ServiceExtensionIsNotAFunctionError) Error() string {
	return fmt.Sprintf("Service extending injection id '%s' is not a function", e.extendedInjectionId)
}

type ServiceFunctionFirstArgumentMustBeAContainerError struct {
	injectionId string
}

func (e ServiceFunctionFirstArgumentMustBeAContainerError) Error() string {
	return fmt.Sprintf("Service '%s' function first argument must be a pointer to the container", e.injectionId)
}

func isFunction(value interface{}) bool {
	return reflect.TypeOf(value).Kind() == reflect.Func
}

func triggerFunctionWithContainer(injectionId string, c *container, function interface{}, args ...interface{}) (interface{}, error) {
	functionReflection := reflect.ValueOf(function)
	functionReflectionType := functionReflection.Type()

	if functionReflectionType.NumIn() < 1 {
		return nil, ServiceFunctionFirstArgumentMustBeAContainerError{injectionId: injectionId}
	}

	functionFirstArgument := functionReflectionType.In(0)
	if functionFirstArgument.Kind() != reflect.Ptr || functionFirstArgument.Elem() == reflect.TypeOf(c) {
		return nil, ServiceFunctionFirstArgumentMustBeAContainerError{injectionId: injectionId}
	}

	functionArgs := []reflect.Value{reflect.ValueOf(c)}
	for i := 0; i < len(args); i++ {
		functionArgs = append(functionArgs, reflect.ValueOf(args[i]))
	}

	return functionReflection.Call(functionArgs)[0].Interface(), nil
}
