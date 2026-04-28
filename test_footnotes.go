package main

import (
"bytes"
"fmt"
"github.com/yuin/goldmark"
"github.com/yuin/goldmark/extension"
)

func main() {
	md := goldmark.New(
goldmark.WithExtensions(extension.Footnote),
)
	var buf bytes.Buffer
	src := []byte(`Hello[^1]

[^1]: World`)
	if err := md.Convert(src, &buf); err != nil {
		panic(err)
	}
	fmt.Println(buf.String())
}
