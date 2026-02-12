# Example: Database-Backed Validation 🗄️

This advanced example demonstrates how to perform Contextual Validation. Unlike standard format checks (like ensuring an email is valid), contextual validation checks business rules against a data source—for example, verifying if an email address is already registered in the database.

## 🧠 The "Injected Closure" Pattern

Since the validator engine doesn't natively support dependency injection, we use a Closure Factory. This allows us to "bake" a database connection (or a mock service) into the validation function before registering it in Transwarp.
🛠️ How it works

  - The Factory: EmailUniqueValidator(db) returns a function that has access to the db variable via closure.

  - Registration: We register this closure under the tag unique_email using middleware.GetValidator().

  - The DTO: Our UserRegistrationDTO uses the tag: validate:"required,email,unique_email".

  - The Middleware: When a request arrives, Transwarp's Validate middleware runs. If the database check fails, it returns a 422 Unprocessable Entity before the handler is ever executed.

## 🚀 Running the Example

1. Start the server

```bash
go run main.go
```
The server will start on http://localhost:8080.


## 🧪 Testing the API (CURL Commands)

The mock database is pre-loaded with: admin@transwarp.io and user@test.com.


✅ Test: New Email (Passes)

This email does not exist in the mock store, so validation will pass.

```Bash
curl -i -X POST http://localhost:8080/register \
     -H "Content-Type: application/json" \
     -d '{
       "email": "dev@transwarp.io",
       "username": "gopher_master"
     }'
```
Expected Status: 200 OK

Expected Body: A success message containing your validated data.


❌ Test: Email Already Taken (Fails)

This request will fail because the email is already in our "database". Transwarp will return a structured error.

```bash
curl -i -X POST http://localhost:8080/register \
     -H "Content-Type: application/json" \
     -d '{
       "email": "admin@transwarp.io",
       "username": "newbie_user"
     }'
```
Expected Status: 422 Unprocessable Entity

Expected Body: `{"status":"error","errors":[{"field":"email","rule":"unique_email","message":"..."}]}`


❌ Test: Invalid Format + Taken Email

Transwarp will catch multiple errors at once (invalid email format and missing username).

```Bash
curl -i -X POST http://localhost:8080/register \
     -H "Content-Type: application/json" \
     -d '{
       "email": "not-an-email",
       "username": ""
     }'
```


## 💡 Why use this in Production?

By moving these checks to the Validation Layer:

  - Handlers stay pure: Your handler logic is reduced to: "The data is valid, so save it."

  - Atomic Errors: You can return all validation errors (format + business rules) in a single HTTP response.

  - Consistency: The same unique_email rule can be reused across different routes (e.g., Update Profile, Registration, etc.).
