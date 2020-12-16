## create Engine
```go
engine := goo.New()
```

## start

```go
engine.SetContext(ctx, wg, nums).Run(url)

-nums: 关闭服务器超时时间
```

## make group
```go
group = engine.Group("group")

group.POST(url, hanler)
group.GET(url, hanler)
...
```

