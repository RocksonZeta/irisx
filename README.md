## irisx
------


### Example
```go
import (
	"github.com/RocksonZeta/irisx"
	"github.com/RocksonZeta/wrap/wraps"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)
func main() {
	app := iris.New()
	sidUidMap := make(map[string]int)
	sidUidMap["abc"] = 100
	irisx.CookieSid2Uid = func(sid string) interface{} {
		return sidUidMap[sid]
	}
	app.ContextPool.Attach(func() context.Context {
		return &irisx.Context{
			Context:  context.NewContext(app),
			PageSize: 20,
		}
	})
	app.Use(irisx.SidFilter)
	app.Get("/", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Ok(c.Uid())
	})
	app.Get("/setuid", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		sidUidMap[c.Sid()] = 123
		c.Ok(c.Uid())
	})
	app.Get("/settoken", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.SetCookieToken("abc", 3600*24*10)
		c.Ok(c.Sid())
	})
	app.Get("/uid", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Ok(c.Uid())
	})

	app.Listen(":9000")
}
```