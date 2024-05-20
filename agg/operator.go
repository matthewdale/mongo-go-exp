package agg

import "go.mongodb.org/mongo-driver/bson"

type Operator bson.D

func Abs(numExpr any) Operator {
	return Operator{{Key: "$abs", Value: numExpr}}
}

func Bottom(outputExpr any, sortBys ...SortBy) Operator {
	return Operator{{
		Key: "$bottom",
		Value: bson.D{{
			Key:   "sortBy",
			Value: sortBysToD(sortBys),
		}, {
			Key:   "output",
			Value: outputExpr,
		}},
	}}
}

func BottomN(outputExpr any, n int64, sortBys ...SortBy) Operator {
	return BottomNExpr(outputExpr, n, sortBys...)
}

func BottomNExpr(outputExpr, nExpr any, sortBys ...SortBy) Operator {
	return Operator{{
		Key: "$bottomN",
		Value: bson.D{{
			Key:   "n",
			Value: nExpr,
		}, {
			Key:   "sortBy",
			Value: sortBysToD(sortBys),
		}, {
			Key:   "output",
			Value: outputExpr,
		}},
	}}
}

func Cond(ifExpr, thenExpr, elseExpr any) Operator {
	return Operator{{
		Key: "$cond",
		Value: bson.D{{
			Key:   "if",
			Value: ifExpr,
		}, {
			Key:   "then",
			Value: thenExpr,
		}, {
			Key:   "else",
			Value: elseExpr,
		}},
	}}
}

func Divide(numeratorExpr, denomExpr any) Operator {
	return Operator{{
		Key:   "$divide",
		Value: bson.A{numeratorExpr, denomExpr},
	}}
}

func Eq(expr1, expr2 any) Operator {
	return Operator{{
		Key:   "$eq",
		Value: bson.A{expr1, expr2},
	}}
}

// TODO: Make a different func for including limit, or pass nil?
func Filter(inputExpr any, as string, condExpr, limitExpr any) Operator {
	body := bson.D{{
		Key:   "input",
		Value: inputExpr,
	}, {
		Key:   "cond",
		Value: condExpr,
	}}
	if len(as) > 0 {
		body = append(body, bson.E{Key: "as", Value: as})
	}
	if limitExpr != nil {
		body = append(body, bson.E{Key: "limit", Value: limitExpr})
	}

	return Operator{{
		Key:   "$filter",
		Value: body,
	}}
}

func In(targetExpr, arrExpr any) Operator {
	return Operator{{
		Key:   "$in",
		Value: bson.A{targetExpr, arrExpr},
	}}
}

func Map(inputExpr any, as string, inExpr any) Operator {
	body := bson.D{{
		Key:   "input",
		Value: inputExpr,
	}}
	if len(as) > 0 {
		body = append(body, bson.E{Key: "as", Value: as})
	}
	// TODO: Why?
	body = append(body, bson.E{Key: "in", Value: inExpr})

	return Operator{{
		Key:   "$map",
		Value: body,
	}}
}

func Max(exprs ...any) Operator {
	// TODO: Why?
	var body any
	if len(exprs) == 1 {
		body = exprs[0]
	} else {
		body = bson.A(exprs)
	}

	return Operator{{
		Key:   "$max",
		Value: body,
	}}
}

func Min(exprs ...any) Operator {
	// TODO: Why?
	var body any
	if len(exprs) == 1 {
		body = exprs[0]
	} else {
		body = bson.A(exprs)
	}

	return Operator{{
		Key:   "$min",
		Value: body,
	}}
}

func MergeObjects(documentExprs ...any) Operator {
	var body any
	// TODO: Why?
	if len(documentExprs) == 1 {
		body = documentExprs[0]
	} else {
		body = bson.A(documentExprs)
	}

	return Operator{{
		Key:   "$mergeObjects",
		Value: body,
	}}
}

func Ne(expr1, expr2 any) Operator {
	return Operator{{
		Key:   "$ne",
		Value: bson.A{expr1, expr2},
	}}
}

func Or(exprs ...any) Operator {
	return Operator{{
		Key:   "$or",
		Value: bson.A(exprs),
	}}
}

func Reduce(inputExpr, initialValueExpr, inExpr any) Operator {
	return Operator{{
		Key: "$reduce",
		Value: bson.D{{
			Key:   "input",
			Value: inputExpr,
		}, {
			Key:   "initialValue",
			Value: initialValueExpr,
		}, {
			Key:   "in",
			Value: inExpr,
		}},
	}}
}

func Sum(numExpr any) Operator {
	return Operator{{Key: "$sum", Value: numExpr}}
}

func Top(outputExpr any, sortBy ...SortBy) Operator {
	return Operator{{
		Key: "$top",
		Value: bson.D{{
			Key:   "sortBy",
			Value: sortBysToD(sortBy),
		}, {
			Key:   "output",
			Value: outputExpr,
		}},
	}}
}

func TopN(outputExpr, n int64, sortBy ...SortBy) Operator {
	return TopNExpr(outputExpr, n, sortBy...)
}

func TopNExpr(outputExpr, nExpr any, sortBy ...SortBy) Operator {
	return Operator{{
		Key: "$topN",
		Value: bson.D{{
			Key:   "n",
			Value: nExpr,
		}, {
			Key:   "sortBy",
			Value: sortBysToD(sortBy),
		}, {
			Key:   "output",
			Value: outputExpr,
		}},
	}}
}
