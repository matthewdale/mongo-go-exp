# mongo-go-exp
Experimental packages for adding functionality to the MongoDB Go Driver.

## Aggregation Pipeline Builders

### Requirements

**Is easier to read and maintain than aggregation pipelines defined with
`bson.D`.**

The `bson.D` syntax for defining large, nested documents like aggregation
pipelines is difficult to read and maintain, mostly because of the huge number
of brackets users must manage. Additionally, getting brackets mismatched can
lead to unexpected BSON marshaling, meaning aggregation pipelines stop working
in difficult-to-diagnose ways.

For example:
?

**Integrates well with existing aggregation pipeline definitions using
`bson.D`.**

We expect many users have existing aggregation pipelines, so we need them to be
able to incrementally migrate their pipeline stages. Additionally, the builder
won't support every pipeline stages, or won't support new stages immediately
after they're added.

For example:
```go
pipeline := []bson.D{
    bson.D{{"$addFields", bson.D{{"newField", "oldField"}}}},
    agg.Sort("newField"),
}
```

Alternatively:
```go
pipeline := agg.Pipeline().
    D(bson.D{{"$addFields", bson.D{{"newField", "oldField"}}}}).
    Sort("newField")
```

The first is preferable because users can replace a stage at a time, rather than
redefine their pipeline and wrap existing stages in method calls.

**The API guides and documents usage, so users don't have to reference the
MongoDB documentation while defining aggregation pipelines.**

MongoDB aggregation pipelines can be difficult to understand for new users.
Understanding what replacement values like `<expression>` or `<accumulator>`
mean often requires reading additional documentation. Users defining aggregation
pipelines using the builder should be able to infer many pipeline stage
parameters by just matching up the types.

For example:
?

**Value flexibility over perfect type safety.**

Users should always be able to use the aggregation builder to define any
aggregation pipeline, even ones that aren't directly supported by the builder
functionality. To accomplish that, some APIs may use the `any` type when there
are a number of different supported types, rather than a more constrained type.