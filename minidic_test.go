package minidic

import (
	"testing"
)

func TestWithString(t *testing.T) {
	c := NewContainer()

	c.add(NewInjection("param", "value"))

	injectedValue := c.get("param")
	if injectedValue != "value" {
		t.Error("Expected \"value\", got ", injectedValue)
	}
}

func TestWithFunction(t *testing.T) {
	c := NewContainer()

	f := func(c *container) service { return service{} }
	c.add(NewInjection("service", f))

	injectedValue := c.get("service")
	_, ok := injectedValue.(service)
	if !ok {
		t.Error("Expected service return value, got ", injectedValue)
	}
}

func TestServicesShouldBeTheSameInstance(t *testing.T) {
	c := NewContainer()

	i := 0
	f := func(c *container) service { i++; return service{i} }
	c.add(NewInjection("service", f))

	injectedValue1 := c.get("service")
	injectedValue2 := c.get("service")
	if injectedValue1 != injectedValue2 {
		t.Error("Expected consecutive calls to the same service to return the same value, got ", injectedValue1, injectedValue2)
	}
}

func TestServicesReceiveAPointerToTheContainer(t *testing.T) {
	c := NewContainer()

	var receivedContainer *container
	f := func(c *container) int { receivedContainer = c; return 33 }
	c.add(NewInjection("service", f))

	c.get("service")
	if receivedContainer != c {
		t.Error("Expected received container pointer to be the same than the one we created, got ", receivedContainer, c)
	}
}

func TestHasInjection(t *testing.T) {
	c := NewContainer()

	c.add(NewInjection("param", "value"))
	c.add(NewInjection("nil", nil))
	f := func(c *container) service { return service{} }
	c.add(NewInjection("service", f))

	if !c.has("param") {
		t.Error("Expected container to contain \"param\"")
	}
	if !c.has("nil") {
		t.Error("Expected container to contain \"nil\"")
	}
	if !c.has("service") {
		t.Error("Expected container to contain \"service\"")
	}
	if c.has("non_existent") {
		t.Error("Expected container to not contain \"non_existent\"")
	}
}

func TestErrorIfGettingNonExistentInjectionId(t *testing.T) {
	c := NewContainer()

	_, err := c.getWithoutPanic("foo")
	if nil == err {
		t.Error("Expected container to return an error for a non-existent injection id when using 'getWithoutPanic()'")
	}
	if _, ok := err.(UnknownInjectionIdError); !ok {
		t.Error("Expected container to return an UnknownInjectionIdError for a non-existent injection id when using 'getWithoutPanic()', got ", err)
	}

	defer func() {
		recoveredErr := recover()
		if nil == recoveredErr {
			t.Error("Expected container to panic for a non-existent injection id when using 'get()'")
		} else {
			if _, ok := recoveredErr.(UnknownInjectionIdError); !ok {
				t.Error("Expected container to panic with an UnknownInjectionIdError for a non-existent injection id when using 'get()', got ", recoveredErr)
			}
		}
	}()
	c.get("foo")
}

func TestInjectionDeletion(t *testing.T) {
	c := NewContainer()

	c.add(NewInjection("param", "value"))

	if !c.has("param") {
		t.Error("Expected container to contain \"param\"")
	}

	e := c.del("param")
	if e != nil {
		t.Error("Expected container to not return an error when deleting an existent injection id")
	}
	if c.has("param") {
		t.Error("Expected container to not contain \"param\" any more")
	}

	e2 := c.del("foo")
	if e2 == nil {
		t.Error("Expected container to return an error when deleting a non-existent injection id")
	}

	e3 := c.del("param")
	if e3 == nil {
		t.Error("Expected container to return an error when deleting a previously deleted injection id")
	}
}

func TestFactoriesShouldReturnDifferentInstancesForEachRetrieval(t *testing.T) {
	c := NewContainer()

	i := 0
	f := func(c *container) service { i++; return service{i} }

	injection := NewInjection("service", f)
	injection.asFactory = true
	c.add(injection)

	injectedValue1 := c.get("service")
	injectedValue2 := c.get("service")
	if injectedValue1 == injectedValue2 {
		t.Error("Expected consecutive calls to the same factory service to return new injections each time, got ", injectedValue1, injectedValue2)
	}
}

func TestServicesDependencies(t *testing.T) {
	c := NewContainer()

	f := func(c *container) string {
		recipient := c.get("recipient")
		if recipientStr, ok := recipient.(string); ok {
			return service{}.sayHi(recipientStr)
		}
		panic(UnknownInjectionIdError{id: "recipient"})
	}

	c.add(NewInjection("recipient", "world"))
	c.add(NewInjection("helloService", f))

	hello := c.get("helloService")
	if helloStr, ok := hello.(string); ok {
		if helloStr != "hello world" {
			t.Error("Expected 'helloService' result to be a 'hello world', got ", helloStr)
		}
	} else {
		t.Error("Expected 'helloService' result to be a string, got ", hello)
	}
}

func TestProtectedFunction(t *testing.T) {
	c := NewContainer()

	f := func() service { return service{33} }
	injection := NewInjection("service", f)
	injection.protected = true
	c.add(injection)

	injectionResult := c.get("service")
	if injectionResult, ok := injectionResult.(func() service); ok {
		serviceValue := injectionResult()
		if serviceValue.id != 33 {
			t.Error("Expected protected service to have id '33', got ", serviceValue.id)
		}
	} else {
		t.Error("Expected protected service to be returned as a function, got ", injectionResult)
	}
}

type service struct {
	id int
}

func (r service) sayHi(recipient string) string {
	return "hello " + recipient
}
