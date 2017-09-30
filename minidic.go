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

func (r *container) add(injection injection) {
	r.injections[injection.injectionId] = injection
}

func (r *container) get(injectionId string) interface{} {
	value, err := r.getWithoutPanic(injectionId)
	if err != nil {
		panic(err)
	}
	return value
}

func (r *container) getWithoutPanic(injectionId string) (interface{}, error) {
	if value, ok := r.functionsResultsCache[injectionId]; ok {
		return *value, nil
	}

	injection, exists := r.injections[injectionId]
	if !exists {
		return nil, UnknownInjectionIdError{id: injectionId}
	}

	value := injection.value

	if isFunction(value) && !injection.protected {
		value = triggerFunctionWithContainer(r, value)
	}

	if !injection.asFactory {
		r.functionsResultsCache[injectionId] = &value
	}

	return value, nil
}

func (r *container) has(injectionId string) bool {
	_, ok := r.injections[injectionId]
	return ok
}

func (r *container) del(injectionId string) error {
	_, ok := r.injections[injectionId]
	if !ok {
		return UnknownInjectionIdError{id: injectionId}
	}

	delete(r.injections, injectionId)

	return nil
}

type UnknownInjectionIdError struct {
	id string
}

func (e UnknownInjectionIdError) Error() string {
	return fmt.Sprintf("Unknown injection id '%s'", e.id)
}

func isFunction(value interface{}) bool {
	return reflect.TypeOf(value).Kind() == reflect.Func
}

func triggerFunctionWithContainer(c *container, function interface{}) interface{} {
	functionArg := []reflect.Value{reflect.ValueOf(c)}
	return reflect.ValueOf(function).Call(functionArg)[0].Interface()
}
