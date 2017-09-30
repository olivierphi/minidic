package minidic

import (
	"fmt"
	"reflect"
	"regexp"
)

// Public API

type Injection interface {
	InjectionId() string
	WithInjectedDependencies(injectionsIds []string) Injection
	MarkAsFactory() Injection
	MarkAsProtected() Injection
}

type Container interface {
	Add(newInjection Injection)
	Get(injectionId string) interface{}
	GetWithoutPanic(injectionId string) (interface{}, error)
	Has(injectionId string) bool
	Del(injectionId string) error
	Extend(injectionId string, function interface{})
}

type injection struct {
	injectionId     string
	value           interface{}
	dependenciesIds []string
	asFactory       bool
	protected       bool
}

// Injection implementation

func NewInjection(injectionId string, value interface{}) Injection {
	return &injection{injectionId: injectionId, value: value}
}

func (i *injection) MarkAsFactory() Injection {
	i.asFactory = true
	return i
}

func (i *injection) MarkAsProtected() Injection {
	i.protected = true
	return i
}

func (i *injection) WithInjectedDependencies(injectionsIds []string) Injection {
	i.dependenciesIds = injectionsIds
	return i
}

func (i *injection) InjectionId() string {
	return i.injectionId
}

type container struct {
	injections            map[string]*injection
	functionsResultsCache map[string]*interface{}
}

// Container implementation

func NewContainer() Container {
	c := new(container)
	c.injections = make(map[string]*injection)
	c.functionsResultsCache = make(map[string]*interface{})
	return c
}

func (c *container) Add(newInjection Injection) {
	if underlyingInjection, ok := newInjection.(*injection); ok {
		c.injections[underlyingInjection.injectionId] = underlyingInjection
	} else {
		panic(fmt.Sprintf("'Container.Add()' argument must be an *injection, got %s", newInjection))
	}
}

func (c *container) Get(injectionId string) interface{} {
	value, err := c.GetWithoutPanic(injectionId)
	if err != nil {
		panic(err)
	}
	return value
}

func (c *container) GetWithoutPanic(injectionId string) (interface{}, error) {
	if value, ok := c.functionsResultsCache[injectionId]; ok {
		return *value, nil
	}

	injection, exists := c.injections[injectionId]
	if !exists {
		return nil, UnknownInjectionIdError{InjectionId: injectionId}
	}

	value := injection.value

	if isFunction(value) && !injection.protected {
		var result interface{}
		var err error
		if injection.dependenciesIds != nil {
			result, err = triggerFunctionWithInjectedIds(injectionId, c, value, injection.dependenciesIds)
		} else {
			result, err = triggerFunctionWithContainer(injectionId, c, value)
		}
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

func (c *container) Has(injectionId string) bool {
	_, ok := c.injections[injectionId]
	return ok
}

func (c *container) Del(injectionId string) error {
	_, ok := c.injections[injectionId]
	if !ok {
		return UnknownInjectionIdError{InjectionId: injectionId}
	}

	delete(c.injections, injectionId)

	return nil
}

func (c *container) Extend(injectionId string, function interface{}) {
	extendedInjection, ok := c.injections[injectionId]
	if !ok {
		panic(UnknownInjectionIdError{InjectionId: injectionId})
	}
	extendedInjectionFunction := extendedInjection.value
	if !isFunction(extendedInjectionFunction) {
		panic(ExtendedServiceIsNotAFunctionError{ExtendedInjectionId: injectionId})
	}
	if !isFunction(function) {
		panic(ServiceExtensionIsNotAFunctionError{ExtendedInjectionId: injectionId})
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

// Errors

type UnknownInjectionIdError struct {
	InjectionId string
}

func (e UnknownInjectionIdError) Error() string {
	return fmt.Sprintf("Unknown injection id '%s'", e.InjectionId)
}

type ExtendedServiceIsNotAFunctionError struct {
	ExtendedInjectionId string
}

func (e ExtendedServiceIsNotAFunctionError) Error() string {
	return fmt.Sprintf("Extended injection id '%s' is not mapped to a function", e.ExtendedInjectionId)
}

type ServiceExtensionIsNotAFunctionError struct {
	ExtendedInjectionId string
}

func (e ServiceExtensionIsNotAFunctionError) Error() string {
	return fmt.Sprintf("Service extending injection id '%s' is not a function", e.ExtendedInjectionId)
}

type ServiceFunctionFirstArgumentMustBeAContainerError struct {
	InjectionId       string
	FirstArgumentType string
}

func (e ServiceFunctionFirstArgumentMustBeAContainerError) Error() string {
	return fmt.Sprintf("Service '%s' function first argument must be a pointer to the container, got ", e.InjectionId, e.FirstArgumentType)
}

// Internal utils

func isFunction(value interface{}) bool {
	return reflect.TypeOf(value).Kind() == reflect.Func
}

func triggerFunctionWithContainer(injectionId string, c *container, function interface{}, args ...interface{}) (result interface{}, err error) {
	functionReflection := reflect.ValueOf(function)
	functionReflectionType := functionReflection.Type()

	if functionReflectionType.NumIn() < 1 {
		err = ServiceFunctionFirstArgumentMustBeAContainerError{injectionId, ""}
		return
	}

	functionArgs := []reflect.Value{reflect.ValueOf(c)}
	for i := 0; i < len(args); i++ {
		functionArgs = append(functionArgs, reflect.ValueOf(args[i]))
	}

	defer func() {
		callErr := recover()
		if callErr == nil {
			return
		}
		if errString, ok := callErr.(string); ok {
			re := regexp.MustCompile("reflect: Call using \\*minidic.container as type (\\w+)")
			if match := re.FindStringSubmatch(errString); match != nil {
				err = ServiceFunctionFirstArgumentMustBeAContainerError{injectionId, match[1]}
				return
			}
		}
		panic(callErr)
	}()

	result = functionReflection.Call(functionArgs)[0].Interface()
	return
}

func triggerFunctionWithInjectedIds(injectionId string, c *container, function interface{}, dependenciesIds []string) (result interface{}, err error) {
	functionReflection := reflect.ValueOf(function)

	functionArgs := []reflect.Value{}
	for i := 0; i < len(dependenciesIds); i++ {
		dependencyId := dependenciesIds[i]
		var resolvedDependency interface{}
		resolvedDependency, err = c.GetWithoutPanic(dependencyId)
		if err != nil {
			return
		}
		functionArgs = append(functionArgs, reflect.ValueOf(resolvedDependency))
	}

	result = functionReflection.Call(functionArgs)[0].Interface()
	return
}
