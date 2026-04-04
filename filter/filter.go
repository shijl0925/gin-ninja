package filter

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/shijl0925/go-toolkits/gormx"
	"gorm.io/gorm/clause"
)

// Operator defines the supported declarative filter operators.
type Operator string

// Combiner defines how multiple tagged fields inside one filter clause are joined.
type Combiner string

const (
	OpEq   Operator = "eq"
	OpNe   Operator = "ne"
	OpGt   Operator = "gt"
	OpGe   Operator = "ge"
	OpLt   Operator = "lt"
	OpLe   Operator = "le"
	OpLike Operator = "like"
	OpIn   Operator = "in"

	CombinerOr Combiner = "or"
)

// Clause is a resolved filter clause from a request input struct.
type Clause struct {
	Field    string
	Fields   []string
	Op       Operator
	Value    any
	Combiner Combiner
}

// Set is a collection of declarative filter clauses.
type Set []Clause

// Parse extracts filter clauses from struct fields tagged with `filter:"field,op"`
// or `filter:"field1|field2,op"` for OR-based multi-field filters.
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
		if err := applyClause(query, clause); err != nil {
			return err
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

		fields, op, combiner, err := parseTag(tag, field)
		if err != nil {
			return err
		}

		value := fieldValue.Interface()
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				continue
			}
			value = fieldValue.Elem().Interface()
		}

		fieldName := fields[0]
		if len(fields) > 1 {
			fieldName = strings.Join(fields, "|")
		}

		*clauses = append(*clauses, Clause{
			Field:    fieldName,
			Fields:   fields,
			Op:       op,
			Value:    value,
			Combiner: combiner,
		})
	}
	return nil
}

func applyClause[T any](query *gormx.Query[T], filterClause Clause) error {
	fields := filterClause.Fields
	if len(fields) == 0 && filterClause.Field != "" {
		fields = []string{filterClause.Field}
	}
	if len(fields) == 0 {
		return fmt.Errorf("filter clause is missing fields")
	}

	if len(fields) == 1 {
		switch filterClause.Op {
		case OpEq:
			query.Eq(fields[0], filterClause.Value)
		case OpNe:
			query.Ne(fields[0], filterClause.Value)
		case OpGt:
			query.Gt(fields[0], filterClause.Value)
		case OpGe:
			query.Ge(fields[0], filterClause.Value)
		case OpLt:
			query.Lt(fields[0], filterClause.Value)
		case OpLe:
			query.Le(fields[0], filterClause.Value)
		case OpLike:
			query.Like(fields[0], filterClause.Value)
		case OpIn:
			query.In(fields[0], filterClause.Value)
		default:
			return fmt.Errorf("unsupported filter operator %q", filterClause.Op)
		}
		return nil
	}

	if filterClause.Combiner != CombinerOr {
		return fmt.Errorf("unsupported filter combiner %q", filterClause.Combiner)
	}

	exprs := make([]clause.Expression, 0, len(fields))
	for _, field := range fields {
		expr, err := buildExpression(field, filterClause.Op, filterClause.Value)
		if err != nil {
			return err
		}
		exprs = append(exprs, expr)
	}
	appendOption(query, gormx.Where(clause.Or(exprs...)))
	return nil
}

func parseTag(tag string, field reflect.StructField) ([]string, Operator, Combiner, error) {
	parts := strings.Split(tag, ",")
	if len(parts) > 2 {
		return nil, "", "", fmt.Errorf("filter tag on %s must be in the form field,op or field1|field2,op", field.Name)
	}

	fieldSpec := strings.TrimSpace(parts[0])
	if fieldSpec == "" {
		return nil, "", "", fmt.Errorf("filter tag on %s must include a field name", field.Name)
	}

	rawFields := strings.Split(fieldSpec, "|")
	fields := make([]string, 0, len(rawFields))
	for _, rawField := range rawFields {
		fieldName := strings.TrimSpace(rawField)
		if fieldName == "" {
			return nil, "", "", fmt.Errorf("filter tag on %s contains an empty field name", field.Name)
		}
		fields = append(fields, fieldName)
	}

	op := OpEq
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		op = Operator(strings.TrimSpace(parts[1]))
	}

	combiner := Combiner("")
	if len(fields) > 1 {
		combiner = CombinerOr
	}

	return fields, op, combiner, nil
}

func buildExpression(field string, op Operator, value any) (clause.Expression, error) {
	column := toColumn(field)

	switch op {
	case OpEq:
		return clause.Eq{Column: column, Value: value}, nil
	case OpNe:
		return clause.Neq{Column: column, Value: value}, nil
	case OpGt:
		return clause.Gt{Column: column, Value: value}, nil
	case OpGe:
		return clause.Gte{Column: column, Value: value}, nil
	case OpLt:
		return clause.Lt{Column: column, Value: value}, nil
	case OpLe:
		return clause.Lte{Column: column, Value: value}, nil
	case OpLike:
		strValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("like filter on %q requires a string value", field)
		}
		return clause.Like{Column: column, Value: "%" + strValue + "%"}, nil
	case OpIn:
		return clause.IN{Column: column, Values: toValues(value)}, nil
	default:
		return nil, fmt.Errorf("unsupported filter operator %q", op)
	}
}

func toColumn(field string) clause.Column {
	column := clause.Column{Name: field}
	if table, name, ok := strings.Cut(field, "."); ok && table != "" && name != "" && !strings.Contains(name, ".") {
		column.Table = table
		column.Name = name
	}
	return column
}

func toValues(value any) []any {
	refValue := reflect.ValueOf(value)
	if !refValue.IsValid() {
		return nil
	}

	if refValue.Kind() == reflect.Slice || refValue.Kind() == reflect.Array {
		values := make([]any, refValue.Len())
		for i := 0; i < refValue.Len(); i++ {
			values[i] = refValue.Index(i).Interface()
		}
		return values
	}

	return []any{value}
}

func appendOption[T any](query *gormx.Query[T], opt gormx.DBOption) {
	optsField := reflect.ValueOf(query).Elem().FieldByName("opts")
	if !optsField.IsValid() {
		panic("filter: gormx.Query no longer exposes internal opts field")
	}

	// gormx.Query does not currently expose a public hook for adding a raw DBOption.
	// Multi-field declarative filters need one grouped WHERE clause, so we append the
	// option directly and fail fast if the upstream query internals ever change.
	writable := reflect.NewAt(optsField.Type(), unsafe.Pointer(optsField.UnsafeAddr())).Elem()
	writable.Set(reflect.Append(writable, reflect.ValueOf(opt)))
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
