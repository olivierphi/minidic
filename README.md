# Minidic

[![Build Status](https://travis-ci.org/DrBenton/minidic.svg?branch=master)](https://travis-ci.org/DrBenton/minidic)

Minidic is a small Dependency Injection Container for the Go language, ported from PHP's [Pimple](https://github.com/silexphp/Pimple/tree/1.1), that consists
of just one file and two public interfaces (about 200 lines of code).

The test suite, and even this README file, are basically a copy-n-paste of Pimple's ones,
with only a light adaptation to Go.

So all kudos go to [Fabien Potencier](http://fabien.potencier.org/) and to Pimple contributors!


Install it:

```bash
    $ go get -u github.com/DrBenton/minidic
```

Then import it in your code, and you're good to go:

```go
    import dic "github.com/DrBenton/minidic"
```

Creating a container is a matter of instating the a `Container` interface:

```go
    container := dic.NewContainer()
```

As many other dependency injection containers, Minidic is able to manage two
different kind of data: *services* and *parameters*.

(note that a quick look at [the test suite](minidic_test.py) can also give you
 a pretty good overview of this module features)

### Defining Parameters

Defining a parameter is as simple as using the Container `Add(newInjection Injection)` method:

```go
    // define some parameters
    container.Add(dic.NewInjection("cookie_name", "SESSION_ID"))
    container.Add(dic.NewInjection("cookie_ttl", 3600))
```

### Defining Services

A service is an object that does something as part of a larger system.
Examples of services: Database connection, templating engine, mailer. Almost
any object could be a service.

Services are defined by functions that return an instance of an object:

```go
    // define some services
    func getSessionStorageConfig(c dic.Container) SessionStorageConfig {
        cookieName := c.Get("cookie_name").(string)
        cookieTTL := c.Get("cookie_ttl").(int)
        return SessionStorageConfig{cookieName, cookieTTL}
    }
    c.Add(dic.NewInjection("session_storage_config", getSessionStorageConfig))

    c.Add(dic.NewInjection("session_storage", func getSessionStorageConfig(c dic.Container) SessionStorage {
        return NewSessionStorage(c.Get("session_storage_config").(SessionStorageConfig))
    })
```

Notice that the function has access to the current container
instance, allowing references to other services or parameters.

As objects are only created when you get them, the order of the definitions
does not matter, and there is no noticeable performance penalty.

Using the defined services is also very easy:

```go
    // get the session storage object
    session := container.Get("session_storage").(SessionStorage)

    // the above call is roughly equivalent to the following code:
    // sessionStorage := SessionStorageConfig{"SESSION_ID", 3600}
    // session := NewSessionStorage(sessionStorage)
```

### Defining Factory Services

By default, each time you get a service, Minidic returns the same instance of it.
If you want a different instance to be returned for all calls, mark the service as being a "factory":

```go
    container.Add(dic.NewInjection("incident_context", generateNewIncidentContext).MarkAsFactory())
```

Now, each call to `container.Get("service")` returns a new instance of the service.

### Protecting Parameters

Because Minidic sees functions as service definitions, you need to
mark the service as being a "protected" one to store them as
parameter:

```go
   container.Add(dic.NewInjection("random_genrator", myRandonGeneratorFunction).MarkAsProtected())
```

### Modifying services after creation

In some cases you may want to modify a service definition after it has been
defined. You can use the `extend()` method to define additional code to
be run on your service just after it is created:

```go
    container.Add(dic.NewInjection("mailer", func (c dic.Container) Mailer {
        return NewMailJetMailer(c.Get("mailjet.login").(string), c.Get("mailjet.password").(string))
    })

    if debug {
        container.Extend("mailer", func(c dic.Container, decoratedMailer Mailer) Mailer {
            return DebugMailer{decoratedMailer, c.Get("app.logs_dir").(string)}
        })
        // in "debug" mode, the "mailer" service is now decorated with a DebugMailer
    }
```

The first argument is the name of the object, the second is a function that
gets access to the decorated object instance and the container, and returns a new
service definition.
