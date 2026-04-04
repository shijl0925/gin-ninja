package filter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/shijl0925/go-toolkits/gormx"
)

// Operator defines the supported declarative filter operators.
type Operator string

const (
	OpEq   Operator = "eq"
	OpNe   Operator = "ne"
	OpGt   Operator = "gt"
	OpGe   Operator = "ge"
	OpLt   Operator = "lt"
	OpLe   Operator = "le"
	OpLike Operator = "like"
	OpIn   Operator = "in"
)

// Clause is a resolved filter clause from a request input struct.
type Clause struct {
	Field string
	Op    Operator
	Value any
}

// Set is a collection of declarative filter clauses.
type Set []Clause

// Parse extracts filter clauses from struct fields tagged with `filter:"field,op"`.
func Parse(input any) (Set, error) {
	var clauses Set
	if err := parseInto(reflect.ValueOf(input), &clauses); err != nil {
		return nil, err
	}
	return clauses, nil
}

// Apply resolves the tagged filter clauses and applies them to a gormx query.
func Apply[T any](query *gormx.Query[T], input any) error {
	if query == nil {
		return nil
	}
	clauses, err := Parse(input)
	if err != nil {
		return err
	}
	for _, clause := range clauses {
		switch clause.Op {
		case OpEq:
			query.Eq(clause.Field, clause.Value)
		case OpNe:
			query.Ne(clause.Field, clause.Value)
		case OpGt:
			query.Gt(clause.Field, clause.Value)
		case OpGe:
			query.Ge(clause.Field, clause.Value)
		case OpLt:
			query.Lt(clause.Field, clause.Value)
		case OpLe:
			query.Le(clause.Field, clause.Value)
		case OpLike:
			query.Like(clause.Field, clause.Value)
		case OpIn:
			query.In(clause.Field, clause.Value)
		default:
			return fmt.Errorf("unsupported filter operator %q", clause.Op)
		}
	}
	return nil
}

func parseInto(value reflect.Value, clauses *Set) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("filter input must be a struct or pointer to struct")
	}

	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		if field.Anonymous {
			if err := parseInto(fieldValue, clauses); err != nil {
				return err
			}
			continue
		}

		tag := strings.TrimSpace(field.Tag.Get("filter"))
		if tag == "" || tag == "-" {
			continue
		}

		if !fieldValue.IsValid() || isEmptyValue(fieldValue) {
			continue
		}

		fieldName, op, err := parseTag(tag, field)
		if err != nil {
			return err
		}

		value := fieldValue.Interface()
		if fieldValue.Kind() == reflect.Ptr {
			value = fieldValue.Elem().Interface()
		}

		*clauses = append(*clauses, Clause{
			Field: fieldName,
			Op:    op,
			Value: value,
		})
	}
	return nil
}

func parseTag(tag string, field reflect.StructField) (string, Operator, error) {
	parts := strings.Split(tag, ",")
	fieldName := strings.TrimSpace(parts[0])
	if fieldName == "" {
		return "", "", fmt.Errorf("filter tag on %s must include a field name", field.Name)
	}

	op := OpEq
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		op = Operator(strings.TrimSpace(parts[1]))
	}
	return fieldName, op, nil
}

func isEmptyValue(value reflect.Value) bool {
	if value.Kind() == reflect.Ptr {
		return value.IsNil()
	}

	switch value.Kind() {
	case reflect.String:
		return value.String() == ""
	case reflect.Slice, reflect.Array, reflect.Map:
		return value.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	default:
		return value.IsZero()
	}
}
