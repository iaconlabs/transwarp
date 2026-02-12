package transwarp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iaconlabs/transwarp"
)

// TestInitialStateCreation verifica que si el request no tiene estado,
// SetStateValue lo crea y lo inicializa correctamente.
func TestInitialStateCreation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	req = transwarp.SetStateValue(req, "key", "value")

	state, ok := transwarp.RequestState(req)
	if !ok {
		t.Fatal("Debería haberse creado un estado nuevo")
	}

	if state.Params["key"] != "value" {
		t.Errorf("Esperado 'value', obtenido '%s'", state.Params["key"])
	}
}

// TestStateSequentialUpdates verifica que múltiples llamadas acumulan valores
// simulando el paso por una cadena de middlewares.
func TestStateSequentialUpdates(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	req = transwarp.SetStateValue(req, "first", "1")
	req = transwarp.SetStateValue(req, "second", "2")

	state, _ := transwarp.RequestState(req)

	if state.Params["first"] != "1" || state.Params["second"] != "2" {
		t.Errorf("Los valores secuenciales no se guardaron correctamente: %+v", state.Params)
	}
}

// TestStateValueOverwrite verifica que asignar una misma clave actualiza el valor anterior.
func TestStateValueOverwrite(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	req = transwarp.SetStateValue(req, "overwrite", "old")
	req = transwarp.SetStateValue(req, "overwrite", "new")

	state, _ := transwarp.RequestState(req)

	if state.Params["overwrite"] != "new" {
		t.Errorf("Esperado 'new', obtenido '%s'", state.Params["overwrite"])
	}
}

// TestRequestStateMissing verifica el comportamiento de RequestState
// cuando se llama sobre un request que nunca pasó por el router o SetStateValue.
func TestRequestStateMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, ok := transwarp.RequestState(req)

	if ok {
		t.Error("RequestState debería devolver ok=false para un request sin contexto de Transwarp")
	}
}

// TestSetStateValuePointers verifica que, aunque el request cambie (por WithContext),
// el mapa interno de Params persiste correctamente.
func TestSetStateValuePointers(t *testing.T) {
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2 := transwarp.SetStateValue(req1, "ptr", "original")

	if req1 == req2 {
		t.Error("SetStateValue debe devolver una nueva instancia de Request debido a WithContext")
	}

	state, _ := transwarp.RequestState(req2)
	if state.Params["ptr"] != "original" {
		t.Error("El valor no se recuperó correctamente del nuevo request")
	}
}

// TestStateValueEmpty verifica que la función acepte valores vacíos
// ya que en HTTP un parámetro puede existir pero no tener contenido.
func TestStateValueEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Caso 1: Llave vacía
	req = transwarp.SetStateValue(req, "", "value_for_no_key")
	// Caso 2: Valor vacío
	req = transwarp.SetStateValue(req, "empty_key", "")

	state, _ := transwarp.RequestState(req)

	if state.Params[""] != "value_for_no_key" {
		t.Errorf("No se guardó correctamente la llave vacía")
	}

	if val, exists := state.Params["empty_key"]; !exists || val != "" {
		t.Errorf("Se esperaba un valor vacío para 'empty_key', obtenido: '%s'", val)
	}
}

// TestStateDuplicateKey verifica que al intentar guardar una llave que ya existe,
// el valor se actualice (comportamiento estándar de un mapa en Go).
func TestStateDuplicateKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	key := "duplicate_key"

	req = transwarp.SetStateValue(req, key, "first_value")
	req = transwarp.SetStateValue(req, key, "second_value")

	state, _ := transwarp.RequestState(req)

	if len(state.Params) != 1 {
		t.Errorf("Se esperaba 1 elemento en el mapa, se encontraron %d", len(state.Params))
	}

	if state.Params[key] != "second_value" {
		t.Errorf("La llave duplicada debería contener el último valor asignado ('second_value')")
	}
}

// TestDeleteStateValue verifica que una llave existente sea removida correctamente.
func TestDeleteStateValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = transwarp.SetStateValue(req, "secret", "12345")

	// Verificamos que existe antes de borrar
	state, _ := transwarp.RequestState(req)
	if _, exists := state.Params["secret"]; !exists {
		t.Fatal("La llave debería existir antes de ser eliminada")
	}

	// Borramos la llave
	req = transwarp.DeleteStateValue(req, "secret")

	// Verificamos que ya no existe
	state, _ = transwarp.RequestState(req)
	if _, exists := state.Params["secret"]; exists {
		t.Error("La llave 'secret' debería haber sido eliminada del estado")
	}
}

// TestDeleteStateValueNonExistent verifica que borrar una llave que no existe
// no provoque pánicos ni errores.
func TestDeleteStateValueNonExistent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = transwarp.SetStateValue(req, "other", "data")

	// Intentamos borrar algo que no está
	req = transwarp.DeleteStateValue(req, "not_found")

	state, _ := transwarp.RequestState(req)
	if len(state.Params) != 1 {
		t.Errorf("El tamaño del mapa debería seguir siendo 1, obtenido: %d", len(state.Params))
	}
}

// TestDeleteStateNoContext verifica que la función sea segura si se llama
// sobre un request que no tiene un estado inicializado.
func TestDeleteStateNoContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// No debe haber pánico aquí
	req = transwarp.DeleteStateValue(req, "any")

	if _, ok := transwarp.RequestState(req); ok {
		t.Error("No debería haberse creado un estado solo por intentar borrar una llave")
	}
}
