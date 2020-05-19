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
	sessionsOptions := irisx.SessionOptions{
		DatabaseUrl: "redis://localhost:50002/0",
		Password:    "hello",
	}
	sessions := irisx.NewSessions(sessionsOptions)
	sessions.GetUid = func(sid string, sidForm int) (interface{}, error) {
		if sid == "vgs6pNYZ7nDKV0U9PANzARzNUkvFg6Cu" { //"JdC5guusjGnJKaOsdk5Tw1+wUghqVg74Vb5ir1nVFp+AZPsV6YW3Eq8dPmFD9xrD"
			return 1, nil
		}
		return 0, nil
	}
	app := iris.New()
	app.ContextPool.Attach(func() context.Context {
		return &irisx.Context{
			Context:  context.NewContext(app),
			PageSize: 20,
		}
	})
	app.Use(sessions.Filter)
	app.Get("/", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Ok(c.Session().GetUidInt())
	})
	app.Get("/setuid", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Session().SetUid(1)
		c.Ok("")
	})
	app.Get("/setuid2", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Session().SetUid(2)
		c.Ok("")
	})
	app.Get("/token", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		sid, token := c.Sessions().NewToken()
		fmt.Println("sid:" + sid)
		c.Ok(token)

	})
	app.Get("/uid", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		uid := c.Session().GetUidInt()
		c.Ok(uid)

	})

	app.Listen(":9000")
}
```