package main

import (
	"fmt"
	"github.com/mailgun/vulcand/Godeps/_workspace/src/github.com/codegangsta/cli"
	log "github.com/mailgun/vulcand/Godeps/_workspace/src/github.com/mailgun/gotools-log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var vulcanUrl string

func main() {
	log.Init([]*log.LogConfig{&log.LogConfig{Name: "console"}})

	app := cli.NewApp()
	app.Name = "vulcanbundle"
	app.Usage = "Command line interface to compile plugins into vulcan binary"
	app.Commands = []cli.Command{
		{
			Name:   "init",
			Usage:  "Init bundle",
			Action: initBundle,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					"middleware, m",
					&cli.StringSlice{},
					"Path to repo and revision, e.g. github.com/mailgun/vulcand-plugins/auth",
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("Error: %s\n", err)
	}
}

func initBundle(c *cli.Context) {
	b, err := NewBundler(c.StringSlice("middleware"))
	if err != nil {
		log.Errorf("Failed to bundle middlewares: %s", err)
		return
	}
	if err := b.bundle(); err != nil {
		log.Errorf("Failed to bundle middlewares: %s", err)
	} else {
		log.Infof("SUCCESS: bundle vulcand and vulcanctl completed")
	}
}

type Bundler struct {
	bundleDir   string
	middlewares []string
}

func NewBundler(middlewares []string) (*Bundler, error) {
	return &Bundler{middlewares: middlewares}, nil
}

func (b *Bundler) bundle() error {
	if err := b.writeTemplates(); err != nil {
		return err
	}
	return nil
}

func (b *Bundler) writeTemplates() error {
	vulcandPath := "."
	packagePath, err := getPackagePath(vulcandPath)
	if err != nil {
		return err
	}

	context := struct {
		Packages    []Package
		PackagePath string
	}{
		Packages:    appendPackages(builtinPackages(), b.middlewares),
		PackagePath: packagePath,
	}

	if err := writeTemplate(
		filepath.Join(vulcandPath, "main.go"), mainTemplate, context); err != nil {
		return err
	}
	if err := writeTemplate(
		filepath.Join(vulcandPath, "registry", "registry.go"), registryTemplate, context); err != nil {
		return err
	}

	if err := writeTemplate(
		filepath.Join(vulcandPath, "vulcanctl", "main.go"), vulcanctlTemplate, context); err != nil {
		return err
	}
	return nil
}

type Package string

func (p Package) Name() string {
	values := strings.Split(string(p), "/")
	return values[len(values)-1]
}

func builtinPackages() []Package {
	return []Package{
		"github.com/mailgun/vulcand/plugin/connlimit",
		"github.com/mailgun/vulcand/plugin/ratelimit",
	}
}

func getPackagePath(dir string) (string, error) {
	path, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	out := strings.Split(path, "src/")
	if len(out) != 2 {
		return "", fmt.Errorf("Failed to locate package path (missing top level src folder)")
	}
	return out[1], nil
}

func appendPackages(in []Package, a []string) []Package {
	for _, p := range a {
		in = append(in, Package(p))
	}
	return in
}

func writeTemplate(filename, contents string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	t, err := template.New(filename).Parse(contents)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return t.Execute(file, data)
}