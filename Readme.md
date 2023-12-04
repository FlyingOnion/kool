# kool

kool is a helper library for creating k8s operators.

## What's new

We use generics! No more `FooInformer`, `BarInformer` or `BazInformer`, just use `kool.Informer`!

So as `Lister` and `Client`.

```go
// Using code-generator or kubebuilder:

var fooInformer FooInformer
var barInformer BarInformer
var bazInformer BazInformer

// Using kool:

import "github.com/FlyingOnion/kool"

var fooInformer kool.Informer[Foo]
var barInformer kool.Informer[Bar]
var bazInformer kool.Informer[Baz]
```

## How to use

We suggest using [koolbuilder](https://flyingonion.github.io/koolbuilder/index.html) to build your operator boilerplate. You don't need to install any code generator binaries.

GitHub Page

https://flyingonion.github.io/koolbuilder/index.html

Or command line

```bash
go get github.com/FlyingOnion/koolbuilder
```