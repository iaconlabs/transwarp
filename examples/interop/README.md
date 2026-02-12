# Example: Middleware Interoperability 🌉

This example showcases the core value of **Transwarp**: the ability to cherry-pick the best features from different Go web frameworks.


In this project, we are running:

1. **Core**: Standard Library `net/http` (via `muxadapter`).
2. **Logging**: Native **Gin** Logger.
3. **Security**: Native **Echo** CORS middleware.

## 🚀 Key Takeaways

- **No Lock-in**: You can move from Gin to Echo (or vice versa) gradually, middleware by middleware.
- **Unified Flow**: Transwarp ensures that the `context.Context` and `Request.Body` survive the transition between these different framework "worlds".
- **Ecosystem Access**: You have access to 100% of the Go web ecosystem, regardless of your chosen router.

## 🏁 Testing

### Check the Log

Run a request and look at your terminal. You will see the classic **Gin-style colorized logs**, even though we aren't using the Gin router!

```bash
curl http://localhost:8080/interop
```



### Check the CORS

Verify that Echo's CORS middleware is working by sending a preflight request:

```Bash
curl -i -X OPTIONS http://localhost:8080/interop \
     -H "Origin: http://example.com" \
     -H "Access-Control-Request-Method: GET"
```


### 💡 Performance Note

While this is powerful, remember that each `From...` bridge adds a small layer of translation. For high-performance hot paths, prioritize native Transwarp middlewares or standard `net/http` middlewares. Use these bridges when you need to reuse complex, battle-tested logic from other frameworks.

