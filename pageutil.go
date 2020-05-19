package irisx

import (
	"io/ioutil"

	"github.com/fatih/structs"
	"github.com/kataras/iris/v12"
)

// param: can be struct or map[string]interface{}
func Render(app iris.Party, path string, html string, param ...interface{}) {
	app.Get(path, func(ctx iris.Context) {
		if len(param) > 0 {
			p := param[0]
			if p != nil {
				var m map[string]interface{}
				if structs.IsStruct(p) {
					m = structs.Map(p)
				} else {
					ok := false
					m, ok = p.(iris.Map)
					if !ok {
						m = p.(map[string]interface{})
					}

				}
				for k, v := range m {
					ctx.ViewData(k, v)
				}
			}
		}
		ctx.View(html)
	})
}
func SendHtml(app iris.Party, path string, html string) error {
	bs, err := ioutil.ReadFile(html)
	app.StaticContent(path, "text/html", bs)
	return err
}

func Rewrite(app iris.Party, from string, to string, statusCode ...int) {
	app.Get(from, func(ctx iris.Context) {
		ctx.Redirect(to, statusCode...)
	})
}
