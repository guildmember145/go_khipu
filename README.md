API de Integración con Khipu en Go

Este proyecto implementa un servidor backend en Go que actúa como intermediario para crear pagos en la plataforma Khipu. La API está diseñada para ser robusta, segura y fácil de configurar, manejando la comunicación con la API real de Khipu, validando datos y gestionando errores de manera estructurada.
Funcionalidades Principales

    Servidor API RESTful: Expone endpoints para interactuar con la lógica de pagos.
    Integración con Khipu: Se comunica directamente con la API de Khipu para crear intenciones de pago.
    Configuración por Entorno: Carga credenciales y configuraciones sensibles de forma segura desde un archivo .env.
    Validación de Datos de Entrada: Comprueba que los datos recibidos (monto, moneda, etc.) sean válidos antes de contactar a Khipu.
    Manejo de Errores Estructurado: Proporciona respuestas de error claras y códigos de estado HTTP apropiados según el tipo de fallo (error de validación, error de Khipu, error del servidor).
    Construcción Dinámica de Payload: Genera el cuerpo de la petición a Khipu de forma inteligente, incluyendo únicamente los campos proporcionados para evitar errores con valores por defecto (como fechas cero).

Tecnologías Utilizadas

    Lenguaje: Go (v1.19+)
    Framework Web: Gin Gonic
    Variables de Entorno: godotenv
    Cliente HTTP: Paquete nativo net/http de Go

1. Configuración del Entorno

Antes de ejecutar la aplicación, es necesario configurar las credenciales de la API de Khipu.

    Crea un archivo llamado .env en la raíz del proyecto.

    Añade las siguientes variables con tus credenciales de la cuenta de desarrollador de Khipu:
    Ini, TOML

    # khipu_api/.env
    KHIPU_MERCHANT_API_KEY="tu-api_key_aqui"
    KHIPU_TARGET_API_URL="https://payment-api.khipu.com"

2. Instalación y Ejecución

Sigue estos pasos para instalar las dependencias y ejecutar el servidor localmente.

    Abre una terminal en la carpeta raíz del proyecto (khipu_api/).

    Instala las dependencias: Ejecuta el siguiente comando para que Go descargue Gin y godotenv.
    Bash

go mod tidy

Ejecuta el servidor: Inicia la aplicación.
Bash

    go run main.go

    Si todo está correcto, verás un mensaje en la consola indicando que el servidor está escuchando en el puerto 8080.

    2025/06/18 13:50:00 Servidor iniciado en el puerto 8080
    [GIN-debug] Listening and serving HTTP on :8080

3. Uso de la API (Ejemplo con Postman)

El endpoint principal para crear un pago es POST /v3/payments.

    Método: POST
    URL: http://localhost:8080/v3/payments

Cabeceras (Headers)

Es fundamental enviar las siguientes cabeceras en la petición:
Clave (Key)	Valor (Value)
Content-Type	application/json
Cuerpo de la Petición (Body)

Envía un objeto JSON en el cuerpo de la petición. Asegúrate de que los valores numéricos como amount no estén entre comillas.

Ejemplo de Petición Exitosa:
JSON

{
    "subject": "Pago por Producto de Prueba",
    "amount": 2500.50,
    "currency": "ARS",
    "transaction_id": "orden-compra-xyz-987",
    "bank_id": "demobank",
    "payer_email": "comprador.feliz@email.com",
    "return_url": "https://mi-tienda.com/pago/gracias",
    "cancel_url": "https://mi-tienda.com/pago/cancelado",
    "notify_url": "https://mi-backend.com/webhooks/khipu"
}

Respuesta Exitosa

Si la petición a Khipu es exitosa, tu API responderá con un código de estado 201 Created y el cuerpo JSON devuelto por Khipu, que contiene las URLs para que el usuario complete el pago.

Ejemplo de Respuesta:
JSON

{
    "payment_id": "khipu_payment_id_real",
    "payment_url": "https://app.khipu.com/payment/web/...",
    "simplified_transfer_url": "https://app.khipu.com/simplified-transfer/...",
    "transfer_url": "https://app.khipu.com/transfer/...",
    "app_url": "khipu:///pos/...",
    "ready_for_terminal": true
}

4. Arquitectura y Lógica Implementada

El código fue refactorizado a partir de los stubs generados por OpenAPI para replicar la lógica robusta de una implementación en Python. Los puntos clave de la arquitectura son:

    Separación de Lógica: La función PostPayment actúa como un controlador, validando la entrada y manejando la respuesta HTTP. La función createKhipuPayment contiene la lógica de servicio, encargándose de la comunicación con Khipu.
    Manejo de Errores Personalizado: Se utiliza un struct de error KhipuServiceError para encapsular errores de forma estructurada, permitiendo devolver códigos de estado y mensajes específicos al cliente.
    Payload Dinámico: Para resolver los errores de validación de Khipu, la lógica no serializa la estructura de datos completa. En su lugar, construye un map[string]interface{} (similar a un diccionario de Python) añadiendo únicamente los campos que tienen un valor válido. Esto evita enviar campos opcionales con valores "cero" (como fechas 0001-01-01) que la API de Khipu rechaza.
