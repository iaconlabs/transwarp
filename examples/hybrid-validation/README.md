# Example: Hybrid Validation 🛰️

This example demonstrates one of Transwarp's most powerful features: Hybrid Binding and Validation.

In modern API design, it is common to receive identifying information (like IDs) in the URL path, while receiving the actual data payload in a JSON body. Usually, this requires manual extraction, manual decoding, and separate validation steps.

Transwarp automates this by merging both data sources into a single, validated Go struct before it even reaches your handler.

## 🛠️ How it Works

  - The Struct: We define a single InventoryUpdateDTO using both param tags (for URL segments) and json tags (for the request body).

  - The Adapter: We use the MuxAdapter with SimpleCleanerMuxConfig to ensure standard Go routing handles dots or complex characters safely.

  - The Middleware: The middleware.Validate interceptor handles the "heavy lifting":

    * It extracts :w_id and :sku from the path.

    * It decodes the JSON body.

    * It runs the validation rules (e.g., ensuring the ID is a valid UUID).

    * It injects the final object into the request context.


## 🚀 Running the Example

1. Enable Local Development

Since this example is part of a monorepo, ensure the local dependencies are linked:

```Bash
$ ./patch_mods.sh on
```Bash


2. Start the Server

Navigate to this directory and run:

```Bash
$ go run main.go
```Bash


The server will start on http://localhost:8080.


## 🧪 Testing the API (CURL Commands)

### ✅ Successful Request

This command provides a valid UUID, a correct 8-character alphanumeric SKU, and a valid JSON body.

```Bash
curl -X POST http://localhost:8080/warehouses/550e8400-e29b-41d4-a716-446655440000/items/PROD1234 \
     -H "Content-Type: application/json" \
     -d '{
       "quantity": 50,
       "reason": "Restock from main hub"
     }'
```


### ❌ Failure: Invalid Path Parameter (UUID)

Try sending an invalid ID. Transwarp will catch this via the validate:"uuid" tag.
Bash

```Bash
curl -i -X POST http://localhost:8080/warehouses/warehouse-1/items/PROD1234 \
     -H "Content-Type: application/json" \
     -d '{"quantity": 10, "reason": "test"}'
```Bash

Expected Result: 422 Unprocessable Entity with an error message regarding the WarehouseID field.


### ❌ Failure: Validation Logic (Quantity)

Try sending a negative quantity. This violates the validate:"min=1" rule.

```Bash
curl -i -X POST http://localhost:8080/warehouses/550e8400-e29b-41d4-a716-446655440000/items/PROD1234 \
     -H "Content-Type: application/json" \
     -d '{"quantity": -5, "reason": "Correction"}'
```

Expected Result: 422 Unprocessable Entity specifically identifying the Quantity field.


### 📝 Key Code Snippet


```Go
// Data is gathered from TWO different sources but handled as ONE unit.
type InventoryUpdateDTO struct {
    WarehouseID string `param:"w_id" validate:"required,uuid"` // From Path
    ItemSKU     string `param:"sku" validate:"required,len=8"` // From Path
    Quantity    int    `json:"quantity" validate:"required,min=1"` // From Body
    Reason      string `json:"reason" validate:"required"`         // From Body
}
```


💡 Why this matters

By using Transwarp's Hybrid Validation, your Handlers stay clean and focused only on business logic. You no longer need to write boi