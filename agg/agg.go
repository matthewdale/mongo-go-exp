package agg

import "go.mongodb.org/mongo-driver/bson"

// TODO: Constrain types?
// TODO: Better name?
type FieldExpr bson.E

func Field(name string, expr any) FieldExpr {
	return FieldExpr{Key: name, Value: expr}
}

type SortBy bson.E

func SortAscending(fieldName string) SortBy {
	return SortBy{Key: fieldName, Value: 1}
}

func SortDescending(fieldName string) SortBy {
	return SortBy{Key: fieldName, Value: -1}
}

func SortExpr(fieldName string, expr any) SortBy {
	return SortBy{Key: fieldName, Value: expr}
}

func sortBysToD(sorts []SortBy) bson.D {
	d := make(bson.D, len(sorts))
	for i := range sorts {
		d[i] = bson.E(sorts[i])
	}
	return d
}
