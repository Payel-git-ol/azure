package azure

// M - сокращение для map[string]interface{}
type M map[string]interface{}

// HTTP статусы
const (
	StatusOK           = 200
	StatusCreated      = 201
	StatusNoContent    = 204
	StatusBadRequest   = 400
	StatusUnauthorized = 401
	StatusNotFound     = 404
	StatusConflict     = 409
	StatusTooManyRequests = 429
	StatusInternalServerError = 500
)
