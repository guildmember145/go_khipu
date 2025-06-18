// khipu_api/openapi/api_default.go
package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// --- Errores Personalizados (equivalente a las clases de excepción en Python) ---

// KhipuServiceError es nuestro error base
type KhipuServiceError struct {
	Message          string
	StatusCode       int
	KhipuResponseData interface{}
}

func (e *KhipuServiceError) Error() string {
	return e.Message
}

// --- Lógica de Servicio (equivalente a khipu_service.py) ---

const (
	defaultKhipuTargetAPIURL = "https://payment-api.khipu.com"
	requestTimeoutSeconds    = 30
)

// prepareKhipuPayload valida los datos de entrada, como en _prepare_khipu_payload en Python
func prepareKhipuPayload(clientData PaymentPostPayment) error {
	if clientData.Subject == "" || clientData.Amount == 0 || clientData.Currency == "" {
		return &KhipuServiceError{
			Message:    "Faltan campos obligatorios: subject, amount, currency",
			StatusCode: http.StatusBadRequest,
		}
	}
	if clientData.Currency != "ARS" {
		return &KhipuServiceError{
			Message:    "Moneda no válida para la API Key configurada. Usar ARS.",
			StatusCode: http.StatusBadRequest,
		}
	}
	if clientData.Amount <= 0 {
		return &KhipuServiceError{
			Message:    "El monto debe ser mayor que cero",
			StatusCode: http.StatusBadRequest,
		}
	}
	return nil
}

// createKhipuPayment es el núcleo, como create_payment_intent en Python
func createKhipuPayment(clientData PaymentPostPayment) (map[string]interface{}, error) {
	// 1. Leer credenciales del entorno
	apiKey := os.Getenv("KHIPU_MERCHANT_API_KEY")
	baseURL := os.Getenv("KHIPU_TARGET_API_URL")
	if baseURL == "" {
		baseURL = defaultKhipuTargetAPIURL
	}

	if apiKey == "" {
		log.Println("Error: KHIPU_MERCHANT_API_KEY no está configurada.")
		return nil, &KhipuServiceError{Message: "Error de configuración del servidor: clave API de Khipu faltante", StatusCode: http.StatusInternalServerError}
	}

	// 2. Validar el payload de entrada
	if err := prepareKhipuPayload(clientData); err != nil {
		return nil, err
	}

	// !!! INICIO DEL CAMBIO IMPORTANTE !!!
	// Creamos un mapa para construir el payload dinámicamente, solo con los campos que tienen valor.
	// Esto imita la lógica de tu khipu_service.py y soluciona el problema de los valores cero.
	payload := make(map[string]interface{})
	
	// Campos obligatorios
	payload["subject"] = clientData.Subject
	payload["amount"] = clientData.Amount
	payload["currency"] = clientData.Currency

	// Campos opcionales: Solo los añadimos si tienen un valor no-cero
	if clientData.TransactionId != "" { payload["transaction_id"] = clientData.TransactionId }
	if clientData.Custom != "" { payload["custom"] = clientData.Custom }
	if clientData.Body != "" { payload["body"] = clientData.Body }
	if clientData.BankId != "" { payload["bank_id"] = clientData.BankId }
	if clientData.PayerEmail != "" { payload["payer_email"] = clientData.PayerEmail }
	if clientData.ReturnUrl != "" { payload["return_url"] = clientData.ReturnUrl }
	if clientData.CancelUrl != "" { payload["cancel_url"] = clientData.CancelUrl }
	if clientData.NotifyUrl != "" { payload["notify_url"] = clientData.NotifyUrl }
	if clientData.PictureUrl != "" { payload["picture_url"] = clientData.PictureUrl }
	
	// Para las fechas, solo las añadimos si no son la "fecha cero" de Go
	if !clientData.ExpiresDate.IsZero() { payload["expires_date"] = clientData.ExpiresDate }
	if !clientData.ConfirmTimeoutDate.IsZero() { payload["confirm_timeout_date"] = clientData.ConfirmTimeoutDate }
	// ... puedes añadir más campos opcionales aquí de la misma forma

	payloadBytes, err := json.Marshal(payload)
	// !!! FIN DEL CAMBIO IMPORTANTE !!!

	if err != nil {
		log.Printf("Error al serializar el payload a JSON: %v", err)
		return nil, &KhipuServiceError{Message: "Error interno al procesar datos", StatusCode: http.StatusInternalServerError}
	}

	log.Printf("JSON enviado a Khipu: %s", string(payloadBytes))

	// 3. Configurar y realizar la petición HTTP (esto sigue igual)
	apiEndpoint := fmt.Sprintf("%s/v3/payments", baseURL)
	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Error al crear la petición HTTP: %v", err)
		return nil, &KhipuServiceError{Message: "Error interno al crear la petición", StatusCode: http.StatusInternalServerError}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: time.Second * requestTimeoutSeconds}
	
	log.Printf("Enviando solicitud a Khipu API: POST %s", apiEndpoint)
	
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error de conexión con Khipu API: %v", err)
		return nil, &KhipuServiceError{Message: fmt.Sprintf("Error de conexión con Khipu API: %s", err.Error()), StatusCode: http.StatusServiceUnavailable}
	}
	defer resp.Body.Close()

	// 4. Procesar la respuesta de Khipu
	body, _ := io.ReadAll(resp.Body)
	log.Printf("Khipu raw response status: %d", resp.StatusCode)
	log.Printf("Khipu raw response body: %s", string(body))
	
	var responseData map[string]interface{}
    if err := json.Unmarshal(body, &responseData); err != nil {
        // Si no se puede decodificar como JSON, se usa el cuerpo como texto plano
        responseData = map[string]interface{}{"raw_response": string(body)}
    }

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("Respuesta exitosa de Khipu.")
		return responseData, nil
	}

	// Si no fue exitosa
	log.Printf("Error HTTP de Khipu: %d", resp.StatusCode)
	
	return nil, &KhipuServiceError{
		Message:          "Error recibido desde la API de Khipu",
		StatusCode:       resp.StatusCode,
		KhipuResponseData: responseData,
	}
}


// --- Implementación de los Handlers de la API (equivalente a payment_routes.py) ---

type DefaultAPI struct{}

// PostPayment es ahora nuestro controlador principal que usa la lógica de servicio.
func (api *DefaultAPI) PostPayment(c *gin.Context) {
	var clientData PaymentPostPayment

	// Validar que la solicitud sea JSON y decodificarla
	if err := c.ShouldBindJSON(&clientData); err != nil {
		log.Printf("Solicitud a /v3/payments no es JSON válido: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "El cuerpo de la solicitud debe ser un JSON válido", "details": err.Error()})
		return
	}

	log.Printf("Ruta /v3/payments recibió datos: %+v", clientData)

	// Llamar a nuestra lógica de servicio para crear el pago
	khipuResponse, err := createKhipuPayment(clientData)

	if err != nil {
		var serviceErr *KhipuServiceError
		// Comprobamos si el error es de nuestro tipo personalizado KhipuServiceError
		if errors.As(err, &serviceErr) {
			log.Printf("Error de servicio Khipu: %s - Data: %v", serviceErr.Message, serviceErr.KhipuResponseData)
			// Devolver el error con los detalles y el código de estado apropiado
			if serviceErr.KhipuResponseData != nil {
				c.JSON(serviceErr.StatusCode, serviceErr.KhipuResponseData)
			} else {
				c.JSON(serviceErr.StatusCode, gin.H{"error": serviceErr.Message})
			}
		} else {
			// Captura cualquier otro error inesperado
			log.Printf("Error inesperado en la creación de pago: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno del servidor al procesar el pago"})
		}
		return
	}

	// Si create_khipu_payment es exitoso, devuelve los datos de Khipu y un 201 Created.
	// ¡Igual que en tu código Python!
	c.JSON(http.StatusCreated, khipuResponse)
}


// --- Mantener los otros endpoints como stubs por ahora ---

func (api *DefaultAPI) DeletePaymentById(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }
func (api *DefaultAPI) GetBanks(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }
func (api *DefaultAPI) GetMerchantPaymentMethodsById(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }
func (api *DefaultAPI) GetPaymentById(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }
func (api *DefaultAPI) GetPredict(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }
func (api *DefaultAPI) PostPaymentConfirmById(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }
func (api *DefaultAPI) PostPaymentRefundsById(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }
func (api *DefaultAPI) PostReceiver(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"status": "Not Implemented"}) }