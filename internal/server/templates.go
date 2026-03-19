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
	"io/fs"
	"log"
	"path/filepath"

	"github.com/metno/go-mms/pkg/assets"
)

// CreateTemplates creates templates from the template files embedded into the binary.
func CreateTemplates() *template.Template {
	templates := template.New("")

	fs.WalkDir(assets.Static, "static/templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if !d.IsDir() {
			templateData, err := assets.Static.ReadFile(path)
			if err != nil {
				log.Fatal(err)
			}
			// Multiple calls to Parse append templates; {{define}} blocks inside each file name the template.
			templates.New(filepath.Base(path)).Parse(string(templateData))
		}
		return nil
	})

	return templates
}
