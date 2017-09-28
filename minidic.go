package minidic

import (
	"fmt"
	"reflect"
)

type container struct {
	values                map[string]interface{}
	callablesResultsCache map[string]*interface{}
	factories             map[string]bool
}

type injection struct {
	injectionId string
	value       interface{}
	asFactory   bool
}

func NewContainer() *container {
	c := new(container)
	c.values = make(map[string]interface{})
	c.callablesResultsCache = make(map[string]*interface{})
	c.factories = make(map[string]bool)
	return c
}

func NewInjection(injectionId string, value interface{}) *injection {
	return &injection{injectionId: injectionId, value: value, asFactory: false}
}

func (r *container) add(injection *injection) {
	r.values[injection.injectionId] = injection.value
	if injection.asFactory {
		r.factories[injection.injectionId] = true
	}
}

func (r *container) get(injectionId string) (interface{}, error) {
	if value, ok := r.callablesResultsCache[injectionId]; ok {
		return *value, nil
	}

	value, exists := r.values[injectionId]
	if !exists {
		return nil, UnknownInjectionIdError{id: injectionId}
	}

	if isFunction(value) {
		functionArg := []reflect.Value{reflect.ValueOf(r)}
		value = reflect.ValueOf(value).Call(functionArg)[0].Interface()
	}

	if !r.factories[injectionId] {
		r.callablesResultsCache[injectionId] = &value
	}

	return value, nil
}

func (r *container) has(injectionId string) bool {
	_, ok := r.values[injectionId]
	return ok
}

func (r *container) del(injectionId string) error {
	_, ok := r.values[injectionId]
	if !ok {
		return UnknownInjectionIdError{id: injectionId}
	}

	delete(r.values, injectionId)

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
