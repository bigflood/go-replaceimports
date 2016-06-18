// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
  "bytes"
	"flag"
  "fmt"
	"go/scanner"
	"io"
  "io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
  "go/parser"
  "go/token"
  "strconv"

  "golang.org/x/tools/go/ast/astutil"
)

var (
	// main operation modes
	list   = flag.Bool("l", false, "list files whose formatting differs from goimport's")
	write  = flag.Bool("w", false, "write result to (source) file instead of stdout")
  doDiff = flag.Bool("d", false, "display diffs instead of rewriting files")
  importPath1 = flag.String("f", "", "import path to find")
	importPath2 = flag.String("r", "", "import path to replace with")

	exitCode = 0
)

func init() {
}

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	exitCode = 2
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: goimports [flags] [path ...]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func isGoFile(f os.FileInfo) bool {
	// ignore non-Go files
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func processFile(filename string, in io.Reader, out io.Writer, stdin bool) error {
	if in == nil {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	res, err := processFileImpl(filename, src)
	if err != nil {
		return err
	}

	if !bytes.Equal(src, res) {
		// formatting has changed
		if *list {
			fmt.Fprintln(out, filename)
		}
		if *write {
			err = ioutil.WriteFile(filename, res, 0)
			if err != nil {
				return err
			}
		}
		if *doDiff {
			data, e := diff(src, res)
			if e != nil {
				return fmt.Errorf("computing diff: %s", e)
			}
			fmt.Printf("diff %s gofmt/%s\n", filename, filename)
			out.Write(data)
		}
	}

	if !*list && !*write && !*doDiff {
		_, err = out.Write(res)
	}

	return err
}

func visitFile(path string, f os.FileInfo, err error) error {
	if err == nil && isGoFile(f) {
		err = processFile(path, nil, os.Stdout, false)
	}
	if err != nil {
		report(err)
	}
	return nil
}

func walkDir(path string) {
	filepath.Walk(path, visitFile)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// call gofmtMain in a separate function
	// so that it can use defer and have them
	// run before the exit.
	gofmtMain()
	os.Exit(exitCode)
}

func gofmtMain() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		if err := processFile("<standard input>", os.Stdin, os.Stdout, true); err != nil {
			report(err)
		}
		return
	}

	for i := 0; i < flag.NArg(); i++ {
		path := flag.Arg(i)
		switch dir, err := os.Stat(path); {
		case err != nil:
			report(err)
		case dir.IsDir():
			walkDir(path)
		default:
			if err := processFile(path, nil, os.Stdout, false); err != nil {
				report(err)
			}
		}
	}
}

func diff(b1, b2 []byte) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "gofmt")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "gofmt")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}

func processFileImpl(filename string, src []byte) ([]byte, error) {
fmt.Printf("Process %v  %v bytes \n", filename, len(src))

	fileSet := token.NewFileSet()

  parserMode := parser.Mode(0)
  parserMode |= parser.ParseComments

  file, err := parser.ParseFile(fileSet, filename, src, parserMode)
	if err != nil {
		return nil, err
	}

	imps := astutil.Imports(fileSet, file)

  type Found struct {
    offset1, offset2 int
  }

  founds := make([]Found, 0, 16)

	for _, impSection := range imps {
		for _, importSpec := range impSection {
			path, _ := strconv.Unquote(importSpec.Path.Value)
      if path != *importPath1 {
        continue
      }

			p1 := fileSet.Position(importSpec.Path.Pos())
			p2 := fileSet.Position(importSpec.Path.End())

      f := Found{offset1:p1.Offset, offset2:p2.Offset}
      founds = append(founds, f)
		}
	}

  if len(founds) > 0 {
    path2 := ([]byte)(fmt.Sprintf("\"%v\"", *importPath2))

    buf := &bytes.Buffer{}

    index := 0

    for _, f := range(founds) {
      if index < f.offset1 {
        buf.Write(src[index:f.offset1])
      }

      buf.Write(path2)

      index = f.offset2
    }

    buf.Write(src[index:])

	   return buf.Bytes(), nil
   }

   return src, nil
 }
