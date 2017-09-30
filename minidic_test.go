package minidic_test

import (
	dic "github.com/DrBenton/minidic"
	"strconv"
	"testing"
)

func TestWithString(t *testing.T) {
	c := dic.NewContainer()

	c.Add(dic.NewInjection("param", "value"))

	injectedValue := c.Get("param")
	if injectedValue != "value" {
		t.Error("Expected \"value\", got ", injectedValue)
	}
}

func TestWithFunction(t *testing.T) {
	c := dic.NewContainer()

	f := func(c dic.Container) service { return service{} }
	c.Add(dic.NewInjection("service", f))

	injectedValue := c.Get("service")
	_, ok := injectedValue.(service)
	if !ok {
		t.Error("Expected service return value, got ", injectedValue)
	}
}

func TestServiceDefinedWithFunctionOnlyArgumentMustBeAContainerOrPanicWillOccur(t *testing.T) {
	c := dic.NewContainer()

	f1 := func() service { return service{} }
	c.Add(dic.NewInjection("service.no_arg", f1))
	f2 := func(test string) service { return service{} }
	c.Add(dic.NewInjection("service.wrong_arg_type", f2))

	defer func() {
		recoveredErr := recover()
		if nil == recoveredErr {
			t.Error("Expected container to panic for a function without arguments")
			return
		} else {
			if _, ok := recoveredErr.(dic.ServiceFunctionFirstArgumentMustBeAContainerError); !ok {
				t.Error("Expected container to panic with an ServiceFunctionFirstArgumentMustBeAContainerError for function without arguments, got ", recoveredErr)
				return
			}
		}

		// ok, let's check "service.wrong_arg_type" now...
		defer func() {
			recoveredErr := recover()
			if nil == recoveredErr {
				t.Error("Expected container to panic for a function which only argument is not a container")
			} else {
				typeError, ok := recoveredErr.(dic.ServiceFunctionFirstArgumentMustBeAContainerError)
				if !ok {
					t.Error("Expected container to panic with an ServiceFunctionFirstArgumentMustBeAContainerError for function which only argument is not a container", recoveredErr)
				}
				if typeError.FirstArgumentType != "string" {
					t.Error("Expected container to panic with an ServiceFunctionFirstArgumentMustBeAContainerError with detected argument type 'string', got ", typeError.FirstArgumentType)
				}
			}
		}()
		c.Get("service.wrong_arg_type")
	}()
	// le't start by a "no arg" check...
	c.Get("service.no_arg")
}

func TestServicesShouldBeTheSameInstance(t *testing.T) {
	c := dic.NewContainer()

	i := 0
	f := func(c dic.Container) service { i++; return service{i} }
	c.Add(dic.NewInjection("service", f))

	injectedValue1 := c.Get("service")
	injectedValue2 := c.Get("service")
	if injectedValue1 != injectedValue2 {
		t.Error("Expected consecutive calls to the same service to return the same value, got ", injectedValue1, injectedValue2)
	}
}

func TestServicesReceiveAPointerToTheContainer(t *testing.T) {
	c := dic.NewContainer()

	var receivedContainer dic.Container
	f := func(c dic.Container) int { receivedContainer = c; return 33 }
	c.Add(dic.NewInjection("service", f))

	c.Get("service")
	if receivedContainer != c {
		t.Error("Expected received container pointer to be the same than the one we created, got ", receivedContainer, c)
	}
}

func TestHasInjection(t *testing.T) {
	c := dic.NewContainer()

	c.Add(dic.NewInjection("param", "value"))
	c.Add(dic.NewInjection("nil", nil))
	f := func(c dic.Container) service { return service{} }
	c.Add(dic.NewInjection("service", f))

	if !c.Has("param") {
		t.Error("Expected container to contain \"param\"")
	}
	if !c.Has("nil") {
		t.Error("Expected container to contain \"nil\"")
	}
	if !c.Has("service") {
		t.Error("Expected container to contain \"service\"")
	}
	if c.Has("non_existent") {
		t.Error("Expected container to not contain \"non_existent\"")
	}
}

func TestErrorIfGettingNonExistentInjectionId(t *testing.T) {
	c := dic.NewContainer()

	_, err := c.GetWithoutPanic("foo")
	if nil == err {
		t.Error("Expected container to return an error for a non-existent injection id when using 'getWithoutPanic()'")
	}
	if _, ok := err.(dic.UnknownInjectionIdError); !ok {
		t.Error("Expected container to return an UnknownInjectionIdError for a non-existent injection id when using 'getWithoutPanic()', got ", err)
	}

	defer func() {
		recoveredErr := recover()
		if nil == recoveredErr {
			t.Error("Expected container to panic for a non-existent injection id when using 'get()'")
		} else {
			if _, ok := recoveredErr.(dic.UnknownInjectionIdError); !ok {
				t.Error("Expected container to panic with an UnknownInjectionIdError for a non-existent injection id when using 'get()', got ", recoveredErr)
			}
		}
	}()
	c.Get("foo")
}

func TestInjectionDeletion(t *testing.T) {
	c := dic.NewContainer()

	c.Add(dic.NewInjection("param", "value"))

	if !c.Has("param") {
		t.Error("Expected container to contain \"param\"")
	}

	e := c.Del("param")
	if e != nil {
		t.Error("Expected container to not return an error when deleting an existent injection id")
	}
	if c.Has("param") {
		t.Error("Expected container to not contain \"param\" any more")
	}

	e2 := c.Del("foo")
	if e2 == nil {
		t.Error("Expected container to return an error when deleting a non-existent injection id")
	}

	e3 := c.Del("param")
	if e3 == nil {
		t.Error("Expected container to return an error when deleting a previously deleted injection id")
	}
}

func TestFactoriesShouldReturnDifferentInstancesForEachRetrieval(t *testing.T) {
	c := dic.NewContainer()

	i := 0
	f := func(c dic.Container) service { i++; return service{i} }

	c.Add(dic.NewInjection("service", f).MarkAsFactory())

	injectedValue1 := c.Get("service")
	injectedValue2 := c.Get("service")
	if injectedValue1 == injectedValue2 {
		t.Error("Expected consecutive calls to the same factory service to return new injections each time, got ", injectedValue1, injectedValue2)
	}
}

func TestServicesDependencies(t *testing.T) {
	c := dic.NewContainer()

	f := func(c dic.Container) string {
		recipient := c.Get("recipient").(string)
		return service{}.sayHi(recipient)
	}

	c.Add(dic.NewInjection("recipient", "world"))
	c.Add(dic.NewInjection("helloService", f))

	hello := c.Get("helloService").(string)
	if hello != "hello world" {
		t.Error("Expected 'helloService' result to be a 'hello world', got ", hello)
	}
}

func TestServicesDependenciesWithAutomaticallyInjectedDependencies(t *testing.T) {
	c := dic.NewContainer()

	c.Add(dic.NewInjection("service", func(c dic.Container) *service { return &service{36} }))
	c.Add(dic.NewInjection("recipient", "world"))
	c.Add(dic.NewInjection("nbExclamationMarks", 3))
	c.Add(dic.NewInjection(
		"helloService",
		func(helloService *service, who string, nbExclamationMarks int) string {
			recipient := who
			for i := 0; i < nbExclamationMarks; i++ {
				recipient += "!"
			}
			recipient += strconv.Itoa(helloService.id)
			return helloService.sayHi(recipient)
		},
	).WithInjectedDependencies([]string{"service", "recipient", "nbExclamationMarks"}))

	hello := c.Get("helloService").(string)
	if hello != "hello world!!!36" {
		t.Error("Expected 'helloService' result to be a 'hello world!!!36', got ", hello)
	}
}

func TestProtectedFunction(t *testing.T) {
	c := dic.NewContainer()

	f := func() service { return service{33} }
	c.Add(dic.NewInjection("service", f).MarkAsProtected())

	injectionResult := c.Get("service")
	if injectionResult, ok := injectionResult.(func() service); ok {
		serviceValue := injectionResult()
		if serviceValue.id != 33 {
			t.Error("Expected protected service to have injectionId '33', got ", serviceValue.id)
		}
	} else {
		t.Error("Expected protected service to be returned as a function, got ", injectionResult)
	}
}

func TestServiceExtension(t *testing.T) {
	c := dic.NewContainer()

	f := func(c dic.Container) service { return service{33} }
	c.Add(dic.NewInjection("service", f))

	c.Extend("service", func(container dic.Container, decoratedServiceResult interface{}) service {
		decoratedService, ok := decoratedServiceResult.(service)
		if !ok {
			t.Error("Expected service decoration first argument to be a service, got ", decoratedServiceResult)
			return service{}
		}
		return service{decoratedService.id * 10}
	})

	injectedValue := c.Get("service")
	serviceValue, ok := injectedValue.(service)
	if !ok {
		t.Error("Expected service return value, got ", injectedValue)
	}
	if serviceValue.id != 330 {
		t.Error("Expected extended service to have injectionId '330', got ", serviceValue.id)
	}
}

type service struct {
	id int
}

func (r service) sayHi(recipient string) string {
	return "hello " + recipient
}
