package querybuilder

import (
	"context"
	"fmt"

	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/huandu/go-sqlbuilder"
)

// SharedCondition is the Compile of every storage without its own condition
// language: the field's read, the shared data-type collision cast, the
// operator, then the guard rule. A sentinel-reading field takes the exists
// guard on the operators that would otherwise match the sentinel.
func SharedCondition(
	ctx context.Context,
	q qbtypes.QueryInfo,
	storage qbtypes.Storage,
	field *telemetrytypes.LogicalField,
	operator qbtypes.FilterOperator,
	value any,
	sb *sqlbuilder.SelectBuilder,
) (qbtypes.Compiled, error) {
	read, err := LogicalRead(ctx, q, storage, field)
	if err != nil {
		return qbtypes.Compiled{}, err
	}
	return SharedConditionForRead(ctx, q, storage, field, read, operator, value, sb)
}

// SharedConditionForRead is SharedCondition over a read the storage built or
// changed itself, for a storage that must rewrite the read before the cast.
func SharedConditionForRead(
	ctx context.Context,
	q qbtypes.QueryInfo,
	storage qbtypes.Storage,
	field *telemetrytypes.LogicalField,
	read qbtypes.Read,
	operator qbtypes.FilterOperator,
	value any,
	sb *sqlbuilder.SelectBuilder,
) (qbtypes.Compiled, error) {
	condition, err := sharedOperator(ctx, q, storage, field, read.SQL, operator, value, sb)
	if err != nil || condition == "" {
		return qbtypes.Compiled{}, err
	}
	if !operator.AddDefaultExistsFilter() || read.WhenAbsent != qbtypes.AbsentIsSentinel {
		return qbtypes.Compiled{Condition: condition}, nil
	}
	return qbtypes.Compiled{Condition: sb.And(condition, sqlbuilder.Escape(read.Presence))}, nil
}

// sharedOperator expands a list per item, so each item takes its own cast,
// and casts the read against the operand for everything else.
func sharedOperator(
	ctx context.Context,
	q qbtypes.QueryInfo,
	storage qbtypes.Storage,
	field *telemetrytypes.LogicalField,
	sql string,
	operator qbtypes.FilterOperator,
	value any,
	sb *sqlbuilder.SelectBuilder,
) (string, error) {
	switch operator {
	case qbtypes.FilterOperatorIn, qbtypes.FilterOperatorNotIn:
		values, ok := value.([]any)
		if !ok {
			return "", qbtypes.ErrInValues
		}
		itemOperator := qbtypes.FilterOperatorEqual
		if operator == qbtypes.FilterOperatorNotIn {
			itemOperator = qbtypes.FilterOperatorNotEqual
		}
		conditions := make([]string, 0, len(values))
		for _, item := range values {
			condition, err := sharedOperator(ctx, q, storage, field, sql, itemOperator, item, sb)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, condition)
		}
		// `=`+OR and `!=`+AND instead of IN and NOT IN, to make use of the index
		if operator == qbtypes.FilterOperatorIn {
			return sb.Or(conditions...), nil
		}
		return sb.And(conditions...), nil
	}

	if operator.IsStringSearchOperator() {
		value = FormatValueForContains(value)
	}
	// Coercion switches only on the data type, which every member shares, so
	// the first member stands in for the field.
	read, value := DataTypeCollisionHandledFieldName(field.Single(), value, sql, operator)
	return OperatorCondition(ctx, q, storage, field, read, operator, value, sb)
}

// OperatorCondition renders one operator over an already cast read. It is
// the shared switch a storage with its own cast policy composes with. A list
// operator is the caller's to expand, so each item takes its own cast.
func OperatorCondition(
	ctx context.Context,
	q qbtypes.QueryInfo,
	storage qbtypes.Storage,
	field *telemetrytypes.LogicalField,
	read string,
	operator qbtypes.FilterOperator,
	value any,
	sb *sqlbuilder.SelectBuilder,
) (string, error) {
	// The read is already a SQL expression produced by the storage. Escape it
	// as SQL rather than passing it to a sqlbuilder comparison helper, which
	// treats both arguments as values and binds the expression as $n.
	fieldExpr := sqlbuilder.Escape(read)
	bind := sb.Var

	switch operator {
	case qbtypes.FilterOperatorEqual:
		return fmt.Sprintf("%s = %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorNotEqual:
		return fmt.Sprintf("%s != %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorGreaterThan:
		return fmt.Sprintf("%s > %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorGreaterThanOrEq:
		return fmt.Sprintf("%s >= %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorLessThan:
		return fmt.Sprintf("%s < %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorLessThanOrEq:
		return fmt.Sprintf("%s <= %s", fieldExpr, bind(value)), nil

	case qbtypes.FilterOperatorLike:
		return fmt.Sprintf("%s LIKE %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorNotLike:
		return fmt.Sprintf("%s NOT LIKE %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorILike:
		return fmt.Sprintf("%s ILIKE %s", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorNotILike:
		return fmt.Sprintf("%s NOT ILIKE %s", fieldExpr, bind(value)), nil

	case qbtypes.FilterOperatorContains:
		return fmt.Sprintf("%s ILIKE %s", fieldExpr, bind(fmt.Sprintf("%%%v%%", value))), nil
	case qbtypes.FilterOperatorNotContains:
		return fmt.Sprintf("%s NOT ILIKE %s", fieldExpr, bind(fmt.Sprintf("%%%v%%", value))), nil

	case qbtypes.FilterOperatorRegexp:
		return fmt.Sprintf("match(%s, %s)", fieldExpr, bind(value)), nil
	case qbtypes.FilterOperatorNotRegexp:
		return fmt.Sprintf("NOT match(%s, %s)", fieldExpr, bind(value)), nil

	case qbtypes.FilterOperatorBetween, qbtypes.FilterOperatorNotBetween:
		values, ok := value.([]any)
		if !ok || len(values) != 2 {
			return "", qbtypes.ErrBetweenValues
		}
		operatorSQL := "BETWEEN"
		if operator == qbtypes.FilterOperatorNotBetween {
			operatorSQL = "NOT BETWEEN"
		}
		return fmt.Sprintf("%s %s %s AND %s", fieldExpr, operatorSQL, bind(values[0]), bind(values[1])), nil

	// exists and not exists are key membership checks, so the storage's
	// presence test answers them
	case qbtypes.FilterOperatorExists, qbtypes.FilterOperatorNotExists:
		logicalRead, err := LogicalRead(ctx, q, storage, field)
		if err != nil {
			return "", err
		}
		if operator == qbtypes.FilterOperatorNotExists {
			return sqlbuilder.Escape(logicalRead.Absence), nil
		}
		return sqlbuilder.Escape(logicalRead.Presence), nil
	}
	return "", qbtypes.ErrUnsupportedOperator
}
