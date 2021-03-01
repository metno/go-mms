/*
Copyright 2020–2021 MET Norway

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	// When this is not a blank import, then VSCode removes it when saving the file
	_ "github.com/metno/go-mms/pkg/statik"
	"github.com/rakyll/statik/fs"
)

// CreateTemplates creates templates from the template files statically built into the binary
func CreateTemplates() *template.Template {
	staticFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	templates := template.New("")

	fs.Walk(staticFS, "/templates", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			templateFile, err := staticFS.Open(path)
			if err != nil {
				log.Fatal(err)
			}

			templateData, err := ioutil.ReadAll(templateFile)
			if err != nil {
				log.Fatal(err)
			}
			// Multiple calls to Parse are appending templates
			templates.New(filepath.Base(path)).Parse(string(templateData))
		}

		return nil
	})

	return templates
}
