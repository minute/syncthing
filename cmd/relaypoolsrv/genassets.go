// Copyright (C) 2015 Audrius Butkevicius and Contributors (see the CONTRIBUTORS file).

// +build ignore

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var tpl = template.Must(template.New("assets").Parse(`package auto

import (
	"encoding/base64"
)

const (
	AssetsBuildDate = "{{.BuildDate}}"
)

func Assets() map[string][]byte {
	var assets = make(map[string][]byte, {{.Assets | len}})
{{range $asset := .Assets}}
	assets["{{$asset.Name}}"], _ = base64.StdEncoding.DecodeString("{{$asset.Data}}"){{end}}
	return assets
}

`))

type asset struct {
	Name string
	Data string
}

var assets []asset

func walkerFor(basePath string) filepath.WalkFunc {
	return func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(filepath.Base(name), ".") {
			// Skip dotfiles
			return nil
		}

		if info.Mode().IsRegular() {
			fd, err := os.Open(name)
			if err != nil {
				return err
			}

			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			io.Copy(gw, fd)
			fd.Close()
			gw.Flush()
			gw.Close()

			name, _ = filepath.Rel(basePath, name)
			assets = append(assets, asset{
				Name: filepath.ToSlash(name),
				Data: base64.StdEncoding.EncodeToString(buf.Bytes()),
			})
		}

		return nil
	}
}

type templateVars struct {
	Assets    []asset
	BuildDate string
}

func main() {
	flag.Parse()

	os.MkdirAll(filepath.Base(flag.Arg(1)), 0777)
	os.Remove(flag.Arg(1))

	filepath.Walk(flag.Arg(0), walkerFor(flag.Arg(0)))
	var buf bytes.Buffer
	tpl.Execute(&buf, templateVars{
		Assets:    assets,
		BuildDate: time.Now().UTC().Format(http.TimeFormat),
	})
	bs, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}

	f, err := os.Create(flag.Arg(1))
	if err != nil {
		panic(err)
	}
	f.Write(bs)
	f.Close()
}
