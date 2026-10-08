package httpx

import(
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type ErrorBody struct{
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct{
	Code string `json:"code"`
	Message string `json:"message"`
	Fields map[string]string `json:"fields,omitempty"`
}

func Init(){
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok{
		v.RegisterTagNameFunc(func(f reflect.StructField) string{
			name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
			if name == "" || name == "-"{
				return f.Name
			}
			return name
		})
	}
}

// OK writes {"data": ....}
func OK(c *gin.Context, status int, data any){
	c.JSON(status, gin.H{"data": data})
}

// Error writes {"error": {"code": ..., "message": ...}}
func Error(c *gin.Context, status int, code, message string){
	c.AbortWithStatusJSON(status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// BindError turns a json-binding failure into a 422(invalid fields) or a 400(malformed json)
func BindError(c *gin.Context, err error){
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs){
		fields := make(map[string]string, len(verrs))
		for _, fe := range verrs{
			fields[fe.Field()] = describe(fe)
		}
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, ErrorBody{Error: ErrorDetail{
			Code: "validation_failed",
			Message: "one or more fields are invalid",
			Fields: fields,
		}})
		return
	}
	Error(c, http.StatusBadRequest, "invalid_body", "request body is not a valid JSON")
}

func describe(fe validator.FieldError)string{
	switch fe.Tag(){
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	case "len":
		return "must be exactly " + fe.Param() + " characters"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "gte":
		return "must be " + fe.Param() + " or more"
	default:
		return "is invalid"
	}
}