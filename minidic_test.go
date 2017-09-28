package minidic

import (
	"testing"
)

func TestWithString(t *testing.T) {
	c := NewContainer()

	c.add(NewInjection("param", "value"))

	injectedValue, _ := c.get("param")
	if injectedValue != "value" {
		t.Error("Expected \"value\", got ", injectedValue)
	}
}

func TestWithFunction(t *testing.T) {
	c := NewContainer()

	f := func(c *container) Service { return Service{} }
	c.add(NewInjection("service", f))

	injectedValue, _ := c.get("service")
	_, ok := injectedValue.(Service)
	if !ok {
		t.Error("Expected service return value, got ", injectedValue)
	}
}

func TestServicesShouldBeTheSameInstance(t *testing.T) {
	c := NewContainer()

	i := 0
	f := func(c *container) Service { i++; return Service{i} }
	c.add(NewInjection("service", f))

	injectedValue1, _ := c.get("service")
	injectedValue2, _ := c.get("service")
	if injectedValue1 != injectedValue2 {
		t.Error("Expected consecutive calls to the same service to return the same value, got ", injectedValue1, injectedValue2)
	}
}

func TestServicesReceiveAPointerToTheContainer(t *testing.T) {
	c := NewContainer()

	var receivedContainer *container
	f := func(c *container) int { receivedContainer = c; return 33 }
	c.add(NewInjection("service", f))

	injectedValue, _ := c.get("service")
	if injectedValue != 33 {
		t.Error("Expected \"33\", got ", injectedValue)
	}
	if receivedContainer != c {
		t.Error("Expected received container pointer to be the same than the one we created, got ", receivedContainer, c)
	}
}

func TestHasInjection(t *testing.T) {
	c := NewContainer()

	c.add(NewInjection("param", "value"))
	c.add(NewInjection("nil", nil))
	f := func(c *container) Service { return Service{} }
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

	_, e := c.get("foo")
	if nil == e {
		t.Error("Expected container to return an error for a non-existent injection id")
	}
	if _, ok := e.(UnknownInjectionIdError); !ok {
		t.Error("Expected container to return an UnknownInjectionIdError for a non-existent injection id, got ", e)
	}
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
	f := func(c *container) Service { i++; return Service{i} }

	injection := NewInjection("service", f)
	injection.asFactory = true
	c.add(injection)

	injectedValue1, _ := c.get("service")
	injectedValue2, _ := c.get("service")
	if injectedValue1 == injectedValue2 {
		t.Error("Expected consecutive calls to the same factory service to return new values each time, got ", injectedValue1, injectedValue2)
	}
}

type Service struct {
	id int
}
