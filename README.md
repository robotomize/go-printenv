## goprintenv

A command line utility that displays the environment variables used by a Go project

### Status

The project is under development, it may not find all environment variables, keep this in mind when using it.

```shell
goprintenv -p <go-project-source>
```

### Install

```shell
go install github.com/robotomize/go-printenv/cmd/goprintenv@latest
```

### Example 

Currently only processes env tags

```go
type Nested struct {
    Live bool `env:"LIVE,default=true"`
}

type NestedStruct struct {
	LastName string `env:"LAST_NAME,default=IVANOV" json:"last_name" bson:"lastName"`
	Two      Nested `env:",prefix=TWO_"`
}

```

The result will be the following

```shell
LAST_NAME=IVANOV
TWO_LIVE=true
```

### @TODO
* add ci, golangci-lint, Makefile, Dockerfile
* Implement parsing of built-in structures
* Should parse os.Getenv
* Support more env tags
* Refactor, simplify the code. Fix bugs and write unit tests