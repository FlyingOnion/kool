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

We provide koolbuilder to generate operator boilerplate online. You don't need to install any code generator binaries.

Choose one of the following links:

Cloudflare Pages (China-mainland-friendly)

https://koolbuilder.pages.dev

GitHub Pages

https://flyingonion.github.io/koolbuilder/index.html

Or use command line (for updating your operator):

```bash
go get github.com/FlyingOnion/koolbuilder
```