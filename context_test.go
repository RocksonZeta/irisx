package irisx_test

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/RocksonZeta/irisx"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

var sessionValues = make(map[string]interface{})

type Sessions struct {
}

func (s *Sessions) GetSessionId(ctx *irisx.Context) string {
	sid := ctx.Values().GetString("__sessionid__")
	if sid != "" {
		return sid
	}
	return ctx.GetCookie("token1")
}
func (s *Sessions) SetSessionId(ctx *irisx.Context) {
	sid := strconv.Itoa(rand.Intn(100000))
	ctx.Values().Set("__sessionid__", sid)
	ctx.SetCookieLocal("token1", sid, 3600, true, "")
}
func (s *Sessions) Set(key string, value interface{}, secs int) error {
	sessionValues[key] = value
	return nil
}
func (s *Sessions) Get(key string, result interface{}) error {
	v, ok := sessionValues[key]
	if !ok {
		return nil
	}
	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(v))
	return nil
}
func (s *Sessions) Refresh(key string, secs int) error {
	return nil
}
func (s *Sessions) Remove(key string) error {
	delete(sessionValues, key)
	return nil
}
func (s *Sessions) UidKey() string {
	return "uid"
}

//go test -run TestContext -v
func TestContext(t *testing.T) {
	app := iris.New()
	app.ContextPool.Attach(func() context.Context {
		return &irisx.Context{
			SessionProvider: &Sessions{},
			Context:         context.NewContext(app),
		}
	})

	app.Use(irisx.SidFilter)
	app.Get("/", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Ok(c.GetUidInt())
	})
	app.Get("/setuid", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.SetUid(10, 3)
		c.Ok(c.GetUidInt())
	})
	app.Get("/setuid2", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.SetUid(2, 3)
		c.Ok(c.GetUidInt())
	})
	app.Get("/token", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Ok(c.Sid())
	})
	app.Get("/uid", func(ctx iris.Context) {
		c := ctx.(*irisx.Context)
		c.Ok(c.GetUidInt())
	})

	app.Listen(":9000")
}
