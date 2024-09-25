package api

import (
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/ProyectoT/api/internal/models"
)

func UpdateFieldAll(litologia interface{}, field string, value interface{}) {
	structValue := reflect.ValueOf(litologia).Elem()
	structField := structValue.FieldByName(field)
	log.Print(structField)

	if !structField.IsValid() {
		fmt.Println("Campo no reconocido:", field)
		return
	}

	if !structField.CanSet() {
		fmt.Println("No se puede establecer el valor del campo:", field)
		return
	}

	val := reflect.ValueOf(value)
	if val.Type().ConvertibleTo(structField.Type()) {
		structField.Set(val.Convert(structField.Type()))
	} else {
		fmt.Printf("Valor inválido para %s: se esperaba %s pero se obtuvo %s\n", field, structField.Type(), val.Type())
		return
	}
}

func GetFieldString(data interface{}, campo string) string {
	val := reflect.ValueOf(data)
	field := val.FieldByName(campo)
	if !field.IsValid() || !field.CanInterface() {
		return ""
	}

	// Usa fmt.Sprintf para convertir el valor del campo a string
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", field.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", field.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", field.Bool())
	default:
		return ""
	}
}

func UpdateFieldLit(litologia *models.LitologiaStruc, field string, value interface{}) {
	switch field {
	case "ColorFill":
		if v, ok := value.(string); ok {
			litologia.ColorFill = v
		} else {
			fmt.Println("Valor inválido para ColorFill")
		}
	case "ColorStroke":
		if v, ok := value.(string); ok {
			litologia.ColorStroke = v
		} else {
			fmt.Println("Valor inválido para ColorStroke")
		}
	case "Zoom":
		if v, ok := value.(int); ok {
			litologia.Zoom = v

		} else if v, err := strconv.Atoi(value.(string)); err == nil {
			litologia.Zoom = v

		} else {
			fmt.Println("Valor inválido para Zoom")
		}
	case "Rotation":
		if v, ok := value.(int); ok {
			litologia.Rotation = v

		} else if v, err := strconv.Atoi(value.(string)); err == nil {
			litologia.Rotation = v

		} else {
			fmt.Println("Valor inválido para Rotation")
		}
	case "Tension":
		if v, ok := value.(float32); ok {
			litologia.Tension = v

		} else if v, err := strconv.ParseFloat(value.(string), 32); err == nil {
			litologia.Tension = float32(v)

		} else {
			fmt.Println("Valor inválido para Tension")
		}
	case "File":
		if v, ok := value.(string); ok {
			litologia.File = v
		} else {
			fmt.Println("Valor inválido para File")
		}
	case "Height":
		if v, ok := value.(int); ok {
			litologia.Height = v
		} else if v, err := strconv.Atoi(value.(string)); err == nil {
			litologia.Height = v
		} else {
			fmt.Println("Valor inválido para Height")
		}
	case "Circles":
		if v, ok := value.([]models.CircleStruc); ok {
			litologia.Circles = v
		} else {
			fmt.Println("Valor inválido para Circles")
		}
	case "Contact":
		if v, ok := value.(string); ok {
			litologia.Contact = v
		} else {
			fmt.Println("Valor inválido para Contact")
		}
	// case "PrevContact":
	// 	if v, ok := value.(string); ok {
	// 		litologia.PrevContact = v
	// 	} else {
	// 		fmt.Println("Valor inválido para PrevContact")
	// 	}
	default:
		fmt.Println("Campo no reconocido")
	}
}
