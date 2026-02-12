# Example: Custom Business Validation 🛡️

While standard tags like `email` or `required` cover 80% of use cases, complex APIs often require custom business rules. This example shows how to hook into the **Transwarp Validation Engine**.

## 🧠 The "Why"

You might want to:

  - Check if a username is already taken in the database.
  - Validate that a SKU follows a specific company pattern.
  - Ensure a date is not on a weekend.

## 🛠️ Implementation

Transwarp uses `go-playground/validator` under the hood. We provide a helper `middleware.GetValidator()` so you can register your own functions:

```go
v := middleware.GetValidator()
v.RegisterValidation("sku_format", myCustomFunc)
```


## 🧪 Testing the Logic

✅ Valid Request (Starts with TW-)

```bash
curl -i -X POST http://localhost:8080/products \
     -H "Content-Type: application/json" \
     -d '{"sku": "TW-BOLT-99", "name": "Titanium Bolt", "price": 12.50}'
```


❌ Invalid Request (Missing TW- prefix)

```Bash
curl -i -X POST http://localhost:8080/products \
     -H "Content-Type: application/json" \
     -d '{"sku": "PROD-001", "name": "Steel Nut", "price": 1.50}'
```
Result: 422 Unprocessable Entity. The error will specifically point to the sku_format tag violation.


### 💡 Pro tip
In a real production environment, you could even inject a **database connection** into your custom validator closure to perform real-time existence checks!
