package agg

import "go.mongodb.org/mongo-driver/bson"

type Stage = bson.D

func AddFields(fields ...FieldExpr) Stage {
	body := make(bson.D, 0, len(fields))
	for _, field := range fields {
		body = append(body, bson.E(field))
	}
	return bson.D{{Key: "$addFields", Value: body}}
}

func Count(fieldName string) Stage {
	return bson.D{{Key: "$count", Value: fieldName}}
}

// TODO: Can we combine these?
func CountAccumulator() Operator {
	return Operator{{Key: "$count", Value: bson.D{}}}
}

func Group(key any, accumulators ...FieldExpr) Stage {
	body := bson.D{{
		Key:   "_id",
		Value: key,
	}}
	for _, acc := range accumulators {
		body = append(body, bson.E(acc))
	}

	return Stage{{
		Key:   "$group",
		Value: body,
	}}
}

// TODO: Make this work with the filter builder?
func Match(query any) Stage {
	return Stage{{Key: "$match", Value: query}}
}

func Project(specifications ...FieldExpr) Stage {
	body := make(bson.D, 0, len(specifications))
	for _, spec := range specifications {
		body = append(body, bson.E(spec))
	}

	return Stage{{
		Key:   "$project",
		Value: body,
	}}
}

func Sort(sortBys ...SortBy) Stage {
	return Stage{{Key: "$sort", Value: sortBysToD(sortBys)}}
}

func Unset(fields ...string) Stage {
	return Stage{{Key: "$unset", Value: fields}}
}

// TODO: Support optional behaviors?
func Unwind(fieldPath string) Stage {
	return Stage{{
		Key: "$unwind",
		Value: bson.D{{
			Key:   "path",
			Value: fieldPath,
		}},
	}}
}
