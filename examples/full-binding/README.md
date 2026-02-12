# Example: Full Triple Binding 🚀

This advanced example showcases **Transwarp's** ability to consolidate data from three distinct HTTP sources into a single Go struct.

## 🎯 The Scenario

Imagine a system where you update product prices. You need:
1. The **Category** from the URL path to ensure the product belongs to a valid section.
2. A **Notify** flag from the query string to decide if customers should be alerted.
3. The **Price** and **Stock** values from the JSON body.

## 🧬 Triple Source Mapping

Transwarp maps these tags automatically:
- `param:"..."` -> URL Path (`/products/:category/...`)
- `query:"..."` -> Query String (`?notify=true`)
- `json:"..."`  -> Request Body (`{"price": 10.5}`)

## 🧪 Testing the Triple Bind

✅ Valid Request

```bash
curl -X POST "http://localhost:8080/products/electronics/update?notify=true" \
     -H "Content-Type: application/json" \
     -d '{"price": 299.99, "stock": 15}'
```


❌ Invalid Category (Validation Error)

```bash
curl -i -X POST "http://localhost:8080/products/food/update?notify=true" \
     -H "Content-Type: application/json" \
     -d '{"price": 5.0, "stock": 10}'
```
Result: Fails because "food" is not in the oneof=electronics books clothing list.


❌ Missing Body Data

```bash
curl -i -X POST "http://localhost:8080/products/books/update" \
     -H "Content-Type: application/json" \
     -d '{"price": 19.99}'
```
Result: Fails because stock is marked as required.




### 💡 Why this is good for Developers

Without Transwarp, you would have to:

1. Use `r.PathValue` for the category.
2. Use `r.URL.Query().Get("notify")` and manually convert it to a boolean.
3. Use `json.Unmarshal` for the body.
4. Manually run a validator on all three.

With Transwarp, you just define the **DTO** and let the middleware do the heavy lifting.

