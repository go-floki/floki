package floki

import (
	"github.com/go-floki/jade"
	"github.com/howeyc/fsnotify"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var templatesData struct {
	compiledTemplates map[string]*template.Template
	directory         string
	compileOptions    jade.Options
}

func (f *Floki) compileTemplates(templatesDir string, logger *log.Logger) map[string]*template.Template {
	var compileOptions jade.Options

	if Env == Prod {
		compileOptions = jade.Options{true, true}
	} else {
		compileOptions = jade.Options{false, false}
	}

	//
	templates, err := jade.CompileDir(templatesDir, jade.DefaultDirOptions, compileOptions)
	if err != nil {
		logger.Printf("Error compiling templates in %s\n", templatesDir)
		panic(err)
	}

	templatesData.compiledTemplates = templates
	templatesData.directory = templatesDir
	templatesData.compileOptions = compileOptions

	if Env == Dev {
		logger.Printf("compiled templates in %s:\n", templatesDir)
		for tplName, _ := range templates {
			logger.Println(tplName)
		}

		logger.Printf("files in %s:\n", templatesDir)
		filepath.Walk(templatesDir, func(path string, f os.FileInfo, err error) error {
			log.Println(path)
			return nil
		})

	}

	watchTemplates := f.Config.Bool("watchTemplates", true)
	if Env == Dev && watchTemplates {
		f.watchTemplates(templatesDir)
	}

	return templates
}

func (f *Floki) watchTemplates(templatesDir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// Process events
	go func() {
		updateTimes := make(map[string]time.Time)

		for {
			select {
			case ev := <-watcher.Event:
				if ev.IsCreate() {
					info, err := os.Stat(ev.Name)
					if err == nil && info.IsDir() {
						log.Println("watching new directory for changes:", ev.Name)

						err = watcher.Watch(ev.Name)
						if err != nil {
							log.Fatal(err)
						}

					}
				}

				if ev.IsDelete() {
					log.Println("removing watch for", ev.Name)
					watcher.RemoveWatch(ev.Name)
				}

				if ev.IsModify() {
					lastUpdated, exists := updateTimes[ev.Name]

					var now time.Time
					info, err := os.Stat(ev.Name)
					if err == nil {
						if exists {
							now = info.ModTime()

							secondsGone := now.Sub(lastUpdated)

							// check if file mtime was updated since last time
							if secondsGone.Seconds() > 0.0 {
								name := strings.Replace(ev.Name, templatesData.directory, "", 1)
								name = strings.Replace(name, ".jade", "", 1)

								log.Println("template updated: ", name)

								/*
									tagsI := f.GetParameter("_tags")
									var tags template.FuncMap
									tags = nil
									if tagsI != nil {
										tags = tagsI.(template.FuncMap)
									}
								*/

								// @todo: build dependencies tree and recompile only needed files
								templates, err := jade.CompileDir(templatesData.directory, jade.DefaultDirOptions, templatesData.compileOptions)
								for k, v := range templates {
									/*if tags != nil {
										v.Funcs(tags)
									}*/

									templatesData.compiledTemplates[k] = v
								}

								//templatesData.compiledTemplates[name], err = jade.CompileFile(ev.Name, templatesData.compileOptions)
								if err != nil {
									log.Fatal(err)
								}

								/* show compiled template for debugging
								 */
								comp := jade.New()
								comp.ParseFile(ev.Name)
								source, _ := comp.CompileString()
								log.Println("compiled:", source)

							}
						}

					} else {
						log.Println(err)
					}

					updateTimes[ev.Name] = now

				}

			case err := <-watcher.Error:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Watch(templatesDir)
	if err != nil {
		log.Fatal(err)
	}

	err = filepath.Walk(templatesDir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			log.Println("watching directory for changes:", path)
			watcher.Watch(path)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("error reading %s directory\n", templatesDir)
	}

	//watcher.Close()
}

func (f *Floki) RegisterTag(tagName string, value interface{}) {
	tagsI := f.GetParameter("_tags")
	if tagsI == nil {
		tags := template.FuncMap{}
		tags[tagName] = value
		f.SetParameter("_tags", tags)
	} else {
		tagsI.(template.FuncMap)[tagName] = value
	}

}
